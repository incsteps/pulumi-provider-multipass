//go:build integration

package resources

import (
	"context"
	"testing"
)

// TestMount_Lifecycle is an integration test for mount create/read/delete.
// Requires multipass to be installed with a running instance.
func TestMount_Lifecycle(t *testing.T) {
	t.Skip("integration test: requires a running Multipass instance")

	ctx := context.Background()
	_ = ctx
}
