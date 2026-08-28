package multipass

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// An explicit TMPDIR must win, so an operator can still direct the file
// somewhere specific.
func TestCloudInitDir_TMPDIRWins(t *testing.T) {
	t.Setenv("TMPDIR", "/some/explicit/dir")
	if got := cloudInitDir(); got != "/some/explicit/dir" {
		t.Errorf("TMPDIR should win, got %q", got)
	}
}

// On Linux the file must land under $HOME, because a snap-confined multipass
// cannot read the host's /tmp and rejects the path with
// "Could not load cloud-init configuration: bad file: /tmp/...".
func TestCloudInitDir_LinuxAvoidsTmp(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("home-directory fallback only applies on linux")
	}
	t.Setenv("TMPDIR", "")
	got := cloudInitDir()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory available")
	}
	if !strings.HasPrefix(got, home) {
		t.Errorf("expected a path under %q so snap confinement can read it, got %q", home, got)
	}
	if want := filepath.Join(home, ".cache", "pulumi-multipass"); got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

// macOS multipass is unconfined, so the default temp dir is correct there and
// returning "" leaves os.CreateTemp on os.TempDir().
func TestCloudInitDir_NonLinuxUsesDefault(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("checks the non-linux branch")
	}
	t.Setenv("TMPDIR", "")
	if got := cloudInitDir(); got != "" {
		t.Errorf("expected \"\" so os.CreateTemp uses os.TempDir(), got %q", got)
	}
}
