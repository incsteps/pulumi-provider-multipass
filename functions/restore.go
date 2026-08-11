package functions

import (
	"context"
	"fmt"

	"github.com/incsteps/pulumi-provider-multipass/internal/multipass"
	"github.com/incsteps/pulumi-provider-multipass/resources"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// RestoreArgs are the inputs for the restore invoke.
type RestoreArgs struct {
	InstanceName       string `pulumi:"instanceName"`
	SnapshotName       string `pulumi:"snapshotName"`
	DestructiveRestore bool   `pulumi:"destructiveRestore,optional"`
}

// RestoreResult is the output of the restore invoke.
type RestoreResult struct {
	Status string `pulumi:"status"` // "ok" on success
}

// Restore is the function controller for restoring a snapshot.
type Restore struct{}

var _ infer.Fn[RestoreArgs, RestoreResult] = (*Restore)(nil)

// Call executes the restore operation.
func (r *Restore) Call(ctx context.Context, input RestoreArgs) (RestoreResult, error) {
	cfg := infer.GetConfig[resources.Config](ctx)
	client := multipass.NewClient(cfg.MultipassBin)

	if err := client.Restore(ctx, multipass.RestoreArgs{
		InstanceName:       input.InstanceName,
		SnapshotName:       input.SnapshotName,
		DestructiveRestore: input.DestructiveRestore,
	}); err != nil {
		return RestoreResult{}, fmt.Errorf("restoring %s from snapshot %s: %w",
			input.InstanceName, input.SnapshotName, err)
	}

	return RestoreResult{Status: "ok"}, nil
}
