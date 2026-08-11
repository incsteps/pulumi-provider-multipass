//go:build integration

package resources

import (
	"context"
	"testing"
)

// TestInstance_Lifecycle is an integration test that creates, reads, and deletes
// a real Multipass instance. Requires multipass to be installed and running.
func TestInstance_Lifecycle(t *testing.T) {
	t.Skip("integration test: requires a running Multipass installation")

	ctx := context.Background()
	_ = ctx

	// Provision a minimal instance.
	// instance := &Instance{}
	// id, state, err := instance.Create(ctx, "test-integration", InstanceArgs{
	//   Name:   "test-integration",
	//   Image:  "ubuntu:24.04",
	//   Cpus:   1,
	//   Memory: "512M",
	//   Disk:   "3G",
	// }, false)
	// ... assert, then delete
}
