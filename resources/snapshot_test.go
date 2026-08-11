//go:build integration

package resources

import (
	"context"
	"testing"
)

// TestSnapshot_Lifecycle is an integration test for snapshot create/read/delete.
// Requires multipass to be installed with a running instance to snapshot.
func TestSnapshot_Lifecycle(t *testing.T) {
	t.Skip("integration test: requires a running Multipass instance")

	ctx := context.Background()
	_ = ctx
}
