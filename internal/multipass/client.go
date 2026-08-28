package multipass

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// execFunc is the function signature used to run an external command.
// It is a field on Client so tests can replace it with a mock.
type execFunc func(ctx context.Context, name string, args ...string) ([]byte, error)

// Client wraps all multipass CLI calls.
type Client struct {
	bin  string   // path to the multipass binary
	exec execFunc // injectable for tests
}

// NewClient returns a Client using the given binary path.
// If bin is empty, "multipass" is looked up in PATH.
func NewClient(bin string) *Client {
	if bin == "" {
		bin = "multipass"
	}
	c := &Client{bin: bin}
	c.exec = c.defaultExec
	return c
}

func (c *Client) defaultExec(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("multipass %s: %w: %s", strings.Join(args, " "), err, errBuf.String())
	}
	return out.Bytes(), nil
}

// cloudInitDir returns a directory the multipass binary can actually read.
//
// On Linux, Multipass is normally installed as a snap, and snap confinement
// denies it access to the host's /tmp. Writing the cloud-init there and passing
// the path to --cloud-init fails with:
//
//	Could not load cloud-init configuration: bad file: /tmp/multipass-cloudinit-*.yaml
//
// The snap does have access to the user's home directory via the `home`
// interface, so prefer a directory under $HOME on Linux. An explicit TMPDIR
// still wins, so a caller can override this. macOS is unaffected, since
// Multipass is not confined there; returning "" leaves os.CreateTemp on its
// default of os.TempDir().
func cloudInitDir() string {
	if d := os.Getenv("TMPDIR"); d != "" {
		return d
	}
	if runtime.GOOS != "linux" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	d := filepath.Join(home, ".cache", "pulumi-multipass")
	if err := os.MkdirAll(d, 0o700); err != nil {
		return ""
	}
	return d
}

// Launch creates and starts a new Multipass instance.
// If args.Cloudinit is non-empty it is written to a temp file and passed via --cloud-init.
func (c *Client) Launch(ctx context.Context, args LaunchArgs) (*InstanceInfo, error) {
	cmdArgs := []string{
		"launch",
		"--name", args.Name,
		"--cpus", fmt.Sprintf("%d", args.Cpus),
		"--memory", args.Memory,
		"--disk", args.Disk,
	}

	var tmpFile string
	if args.Cloudinit != "" {
		f, err := os.CreateTemp(cloudInitDir(), "multipass-cloudinit-*.yaml")
		if err != nil {
			return nil, fmt.Errorf("creating cloud-init temp file: %w", err)
		}
		tmpFile = f.Name()
		if _, err := f.WriteString(args.Cloudinit); err != nil {
			f.Close()
			os.Remove(tmpFile)
			return nil, fmt.Errorf("writing cloud-init temp file: %w", err)
		}
		f.Close()
		defer os.Remove(tmpFile)
		cmdArgs = append(cmdArgs, "--cloud-init", tmpFile)
	}

	// Allow up to 15 minutes for cloud-init to complete. The default Multipass
	// timeout (5 min) is too short when cloud-init installs large packages (JDK,
	// Apptainer, Docker images).
	cmdArgs = append(cmdArgs, "--timeout", "900")

	if args.Image != "" {
		cmdArgs = append(cmdArgs, args.Image)
	}

	if _, err := c.exec(ctx, c.bin, cmdArgs...); err != nil {
		return nil, err
	}

	return c.Info(ctx, args.Name)
}

// Info returns the InstanceInfo for the named instance, or (nil, nil) if not found.
func (c *Client) Info(ctx context.Context, name string) (*InstanceInfo, error) {
	out, err := c.exec(ctx, c.bin, "info", "--format", "json", name)
	if err != nil {
		// multipass exits non-zero when the instance is not found.
		if strings.Contains(err.Error(), "does not exist") ||
			strings.Contains(err.Error(), "not found") ||
			strings.Contains(err.Error(), "No such file") {
			return nil, nil
		}
		return nil, err
	}

	var info InfoOutput
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, fmt.Errorf("parsing multipass info output: %w", err)
	}

	inst, ok := info.Info[name]
	if !ok {
		return nil, nil
	}
	return inst, nil
}

// Stop halts a running instance. Idempotent — returns nil if already stopped.
func (c *Client) Stop(ctx context.Context, name string) error {
	_, err := c.exec(ctx, c.bin, "stop", name)
	if err != nil {
		if strings.Contains(err.Error(), "already stopped") ||
			strings.Contains(err.Error(), "does not exist") {
			return nil
		}
		return err
	}
	return nil
}

// Start boots a stopped instance.
func (c *Client) Start(ctx context.Context, name string) error {
	_, err := c.exec(ctx, c.bin, "start", name)
	return err
}

// Delete removes the named instance. Idempotent — returns nil if instance is absent.
func (c *Client) Delete(ctx context.Context, name string) error {
	_, err := c.exec(ctx, c.bin, "delete", name)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") ||
			strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}
	return nil
}

// Purge purges all deleted instances.
func (c *Client) Purge(ctx context.Context) error {
	_, err := c.exec(ctx, c.bin, "purge")
	return err
}

// Snapshot creates a snapshot of the named instance.
func (c *Client) Snapshot(ctx context.Context, args SnapshotArgs) error {
	cmdArgs := []string{"snapshot", args.InstanceName, "--name", args.SnapshotName}
	if args.Comment != "" {
		cmdArgs = append(cmdArgs, "--comment", args.Comment)
	}
	_, err := c.exec(ctx, c.bin, cmdArgs...)
	return err
}

// InfoSnapshots returns the snapshot list for an instance.
// Returns nil, nil if the instance is not found.
func (c *Client) InfoSnapshots(ctx context.Context, instanceName string) ([]SnapshotInfo, error) {
	out, err := c.exec(ctx, c.bin, "info", "--snapshots", "--format", "json", instanceName)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") ||
			strings.Contains(err.Error(), "not found") {
			return nil, nil
		}
		return nil, err
	}

	var info InfoOutput
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, fmt.Errorf("parsing multipass snapshots output: %w", err)
	}

	inst, ok := info.Info[instanceName]
	if !ok {
		return nil, nil
	}

	// Flatten map[name]*SnapshotInfo → []SnapshotInfo, injecting the name.
	snaps := make([]SnapshotInfo, 0, len(inst.Snapshots))
	for name, s := range inst.Snapshots {
		if s == nil {
			continue
		}
		s.Name = name
		snaps = append(snaps, *s)
	}
	return snaps, nil
}

// DeleteSnapshot deletes a named snapshot from an instance.
// Multipass uses dot notation: `multipass delete instance.snapshot`
// Idempotent — returns nil if snapshot or instance is absent.
func (c *Client) DeleteSnapshot(ctx context.Context, instanceName, snapshotName string) error {
	// --purge is required: Multipass refuses to delete snapshots without it
	// (they cannot sit in a "trashed" state like instances can).
	_, err := c.exec(ctx, c.bin, "delete", "--purge", instanceName+"."+snapshotName)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") ||
			strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}
	return nil
}

// Mount mounts a host directory into an instance.
func (c *Client) Mount(ctx context.Context, args MountArgs) error {
	cmdArgs := []string{"mount"}
	if args.MountType != "" {
		cmdArgs = append(cmdArgs, "--type", args.MountType)
	}
	cmdArgs = append(cmdArgs, args.SourcePath, fmt.Sprintf("%s:%s", args.InstanceName, args.TargetPath))
	_, err := c.exec(ctx, c.bin, cmdArgs...)
	return err
}

// Umount unmounts a path from an instance. Idempotent.
func (c *Client) Umount(ctx context.Context, instanceName, targetPath string) error {
	_, err := c.exec(ctx, c.bin, "umount", fmt.Sprintf("%s:%s", instanceName, targetPath))
	if err != nil {
		if strings.Contains(err.Error(), "not mounted") ||
			strings.Contains(err.Error(), "does not exist") ||
			strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}
	return nil
}

// Restore restores an instance from a snapshot.
func (c *Client) Restore(ctx context.Context, args RestoreArgs) error {
	cmdArgs := []string{"restore", args.InstanceName, "--snapshot", args.SnapshotName}
	if args.DestructiveRestore {
		cmdArgs = append(cmdArgs, "--destructive")
	}
	_, err := c.exec(ctx, c.bin, cmdArgs...)
	return err
}
