package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/incsteps/pulumi-provider-multipass/internal/multipass"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// SnapshotArgs are the inputs for the Snapshot resource.
type SnapshotArgs struct {
	InstanceName string `pulumi:"instanceName"`
	SnapshotName string `pulumi:"snapshotName"`
	Comment      string `pulumi:"comment,optional"`
}

// SnapshotState is the full output state for the Snapshot resource.
type SnapshotState struct {
	SnapshotArgs
	CreatedAt string `pulumi:"createdAt"`
	Parent    string `pulumi:"parent"`
	Size      string `pulumi:"size"`
}

// Snapshot is the resource controller for Multipass VM snapshots.
type Snapshot struct{}

var _ infer.CustomResource[SnapshotArgs, SnapshotState] = (*Snapshot)(nil)
var _ infer.CustomRead[SnapshotArgs, SnapshotState] = (*Snapshot)(nil)
var _ infer.CustomDelete[SnapshotState] = (*Snapshot)(nil)
var _ infer.CustomDiff[SnapshotArgs, SnapshotState] = (*Snapshot)(nil)
var _ infer.Annotated = (*Snapshot)(nil)

func (s *Snapshot) Annotate(a infer.Annotator) {
	a.Describe(s, "A snapshot of a Multipass virtual machine instance.")
}

// Create takes a snapshot. On macOS (QEMU backend) Multipass requires the
// instance to be stopped first; on Linux (LXD) live snapshots are supported.
// We stop → snapshot → restart so the resource works on both platforms.
func (s *Snapshot) Create(
	ctx context.Context, name string, inputs SnapshotArgs, preview bool,
) (string, SnapshotState, error) {
	id := inputs.InstanceName + "." + inputs.SnapshotName
	if preview {
		return id, SnapshotState{SnapshotArgs: inputs}, nil
	}

	cfg := infer.GetConfig[Config](ctx)
	client := multipass.NewClient(cfg.MultipassBin)

	// Check current state — only stop if Running.
	info, err := client.Info(ctx, inputs.InstanceName)
	if err != nil {
		return "", SnapshotState{}, fmt.Errorf("checking instance state before snapshot: %w", err)
	}
	wasRunning := info != nil && info.State == "Running"

	if wasRunning {
		if err := client.Stop(ctx, inputs.InstanceName); err != nil {
			return "", SnapshotState{}, fmt.Errorf("stopping instance %s before snapshot: %w",
				inputs.InstanceName, err)
		}
	}

	snapErr := client.Snapshot(ctx, multipass.SnapshotArgs{
		InstanceName: inputs.InstanceName,
		SnapshotName: inputs.SnapshotName,
		Comment:      inputs.Comment,
	})

	// Always restart if we stopped it, even on snapshot failure.
	if wasRunning {
		if startErr := client.Start(ctx, inputs.InstanceName); startErr != nil {
			// Log start failure but surface snapshot error if both failed.
			if snapErr != nil {
				return "", SnapshotState{}, fmt.Errorf("snapshot failed (%w); also failed to restart: %v",
					snapErr, startErr)
			}
			return "", SnapshotState{}, fmt.Errorf("snapshot succeeded but failed to restart %s: %w",
				inputs.InstanceName, startErr)
		}
	}

	if snapErr != nil {
		// Idempotent create: adopt an existing snapshot rather than failing.
		// This handles ~provider diffs that trigger replacement during development.
		if strings.Contains(snapErr.Error(), "already exists") {
			snapErr = nil
		} else {
			return "", SnapshotState{}, fmt.Errorf("creating snapshot %s on %s: %w",
				inputs.SnapshotName, inputs.InstanceName, snapErr)
		}
	}

	snaps, err := client.InfoSnapshots(ctx, inputs.InstanceName)
	if err != nil {
		return "", SnapshotState{}, err
	}

	state := SnapshotState{SnapshotArgs: inputs}
	for _, snap := range snaps {
		if snap.Name == inputs.SnapshotName {
			state.CreatedAt = snap.Created
			state.Parent = snap.Parent
			state.Size = snap.Size
			break
		}
	}

	return id, state, nil
}

// Read refreshes snapshot state.
func (s *Snapshot) Read(
	ctx context.Context, id string, inputs SnapshotArgs, state SnapshotState,
) (string, SnapshotArgs, SnapshotState, error) {
	cfg := infer.GetConfig[Config](ctx)
	client := multipass.NewClient(cfg.MultipassBin)

	snaps, err := client.InfoSnapshots(ctx, inputs.InstanceName)
	if err != nil {
		return "", inputs, state, fmt.Errorf("reading snapshots for %s: %w", inputs.InstanceName, err)
	}

	for _, snap := range snaps {
		if snap.Name == inputs.SnapshotName {
			newState := SnapshotState{
				SnapshotArgs: inputs,
				CreatedAt:    snap.Created,
				Parent:       snap.Parent,
				Size:         snap.Size,
			}
			return id, inputs, newState, nil
		}
	}

	// Not found — signal by returning empty ID.
	return "", inputs, SnapshotState{}, nil
}

// Delete removes a snapshot.
// On macOS (QEMU) the instance must be stopped before snapshot deletion;
// we stop → delete → restart, mirroring the Create behaviour.
func (s *Snapshot) Delete(ctx context.Context, id string, props SnapshotState) error {
	cfg := infer.GetConfig[Config](ctx)
	client := multipass.NewClient(cfg.MultipassBin)

	info, err := client.Info(ctx, props.InstanceName)
	if err != nil {
		return fmt.Errorf("checking instance state before snapshot delete: %w", err)
	}
	wasRunning := info != nil && info.State == "Running"

	if wasRunning {
		if err := client.Stop(ctx, props.InstanceName); err != nil {
			return fmt.Errorf("stopping instance %s before snapshot delete: %w",
				props.InstanceName, err)
		}
	}

	delErr := client.DeleteSnapshot(ctx, props.InstanceName, props.SnapshotName)

	if wasRunning {
		if startErr := client.Start(ctx, props.InstanceName); startErr != nil {
			if delErr != nil {
				return fmt.Errorf("snapshot delete failed (%w); also failed to restart: %v",
					delErr, startErr)
			}
			return fmt.Errorf("snapshot deleted but failed to restart %s: %w",
				props.InstanceName, startErr)
		}
	}

	if delErr != nil {
		return fmt.Errorf("deleting snapshot %s on %s: %w",
			props.SnapshotName, props.InstanceName, delErr)
	}
	return nil
}

// Diff marks all input fields as requiring replacement.
func (s *Snapshot) Diff(
	ctx context.Context, id string, olds SnapshotState, news SnapshotArgs,
) (p.DiffResponse, error) {
	diff := map[string]p.PropertyDiff{}
	replaceFields := map[string]bool{
		"instanceName": olds.InstanceName != news.InstanceName,
		"snapshotName": olds.SnapshotName != news.SnapshotName,
		"comment":      olds.Comment != news.Comment,
	}

	hasChanges := false
	for field, changed := range replaceFields {
		if changed {
			diff[field] = p.PropertyDiff{Kind: p.UpdateReplace}
			hasChanges = true
		}
	}

	// DeleteBeforeReplace: snapshot IDs are "<instance>.<snapshot>" — a snapshot
	// with the same name on the same instance cannot be created while the old one
	// exists. Pulumi's default create-before-delete order would fail or adopt the
	// old snapshot and then delete it as the "original".
	return p.DiffResponse{
		DeleteBeforeReplace: hasChanges,
		HasChanges:          hasChanges,
		DetailedDiff:        diff,
	}, nil
}
