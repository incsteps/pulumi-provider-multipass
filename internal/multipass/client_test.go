package multipass

import (
	"context"
	"fmt"
	"testing"
)

// newTestClient returns a Client whose exec function is replaced with the provided mock.
func newTestClient(mock execFunc) *Client {
	c := NewClient("")
	c.exec = mock
	return c
}

func TestInfo_Running(t *testing.T) {
	payload := `{
		"info": {
			"my-vm": {
				"state": "Running",
				"ipv4": ["192.168.64.5"],
				"image": {"id": "sha256:abc123", "release": "24.04 LTS"},
				"mounts": {},
				"snapshots": {}
			}
		}
	}`

	c := newTestClient(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(payload), nil
	})

	info, err := c.Info(context.Background(), "my-vm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil InstanceInfo")
	}
	if info.State != "Running" {
		t.Errorf("expected state Running, got %s", info.State)
	}
	if len(info.IPv4) != 1 || info.IPv4[0] != "192.168.64.5" {
		t.Errorf("unexpected IPv4: %v", info.IPv4)
	}
	if info.Image == nil || info.Image.ID != "sha256:abc123" {
		t.Errorf("unexpected image: %v", info.Image)
	}
}

func TestInfo_Stopped(t *testing.T) {
	payload := `{
		"info": {
			"stopped-vm": {
				"state": "Stopped",
				"ipv4": [],
				"image": {"id": "sha256:def456", "release": "22.04 LTS"},
				"mounts": {},
				"snapshots": {}
			}
		}
	}`

	c := newTestClient(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(payload), nil
	})

	info, err := c.Info(context.Background(), "stopped-vm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil InstanceInfo")
	}
	if info.State != "Stopped" {
		t.Errorf("expected state Stopped, got %s", info.State)
	}
}

func TestInfo_NotFound(t *testing.T) {
	c := newTestClient(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, fmt.Errorf("multipass info: exit status 1: instance does not exist")
	})

	info, err := c.Info(context.Background(), "missing-vm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Errorf("expected nil InstanceInfo for missing VM, got %+v", info)
	}
}

func TestInfoSnapshots_Parsed(t *testing.T) {
	payload := `{
		"info": {
			"my-vm": {
				"state": "Running",
				"ipv4": ["192.168.64.5"],
				"image": {"id": "sha256:abc", "release": "24.04 LTS"},
				"mounts": {},
				"snapshots": {
					"post-init": {
						"comment": "initial setup done",
						"parent": "",
						"created": "2024-01-15T10:00:00Z",
						"size": "512.0MiB"
					}
				}
			}
		}
	}`

	c := newTestClient(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(payload), nil
	})

	snaps, err := c.InfoSnapshots(context.Background(), "my-vm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snaps) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snaps))
	}
	s := snaps[0]
	if s.Name != "post-init" {
		t.Errorf("expected snapshot name post-init, got %s", s.Name)
	}
	if s.Comment != "initial setup done" {
		t.Errorf("unexpected comment: %s", s.Comment)
	}
	if s.Size != "512.0MiB" {
		t.Errorf("unexpected size: %s", s.Size)
	}
}

func TestInfoSnapshots_NotFound(t *testing.T) {
	c := newTestClient(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, fmt.Errorf("multipass info: exit status 1: instance not found")
	})

	snaps, err := c.InfoSnapshots(context.Background(), "missing-vm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snaps != nil {
		t.Errorf("expected nil for missing instance, got %v", snaps)
	}
}

func TestDelete_Idempotent(t *testing.T) {
	c := newTestClient(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, fmt.Errorf("multipass delete: exit status 1: instance does not exist")
	})

	if err := c.Delete(context.Background(), "missing-vm"); err != nil {
		t.Errorf("expected nil error for already-deleted VM, got %v", err)
	}
}

func TestDelete_Error(t *testing.T) {
	c := newTestClient(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, fmt.Errorf("connection refused")
	})

	if err := c.Delete(context.Background(), "my-vm"); err == nil {
		t.Error("expected error for non-idempotent failure, got nil")
	}
}
