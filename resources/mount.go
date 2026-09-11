package resources

import (
	"context"
	"fmt"

	"github.com/incsteps/pulumi-provider-multipass/internal/multipass"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// MountArgs are the inputs for the Mount resource.
type MountArgs struct {
	InstanceName string `pulumi:"instanceName"`
	SourcePath   string `pulumi:"sourcePath"`
	TargetPath   string `pulumi:"targetPath"`
	MountType    string `pulumi:"mountType,optional"`
}

// MountState is the full output state for the Mount resource.
type MountState struct {
	MountArgs
}

// Mount is the resource controller for Multipass directory mounts.
type Mount struct{}

var _ infer.CustomResource[MountArgs, MountState] = (*Mount)(nil)
var _ infer.CustomRead[MountArgs, MountState] = (*Mount)(nil)
var _ infer.CustomDelete[MountState] = (*Mount)(nil)
var _ infer.CustomDiff[MountArgs, MountState] = (*Mount)(nil)
var _ infer.Annotated = (*Mount)(nil)
var _ infer.Annotated = (*MountArgs)(nil)

func (m *Mount) Annotate(a infer.Annotator) {
	a.Describe(m, "A host directory mount into a Multipass virtual machine instance.")
}

func (a *MountArgs) Annotate(ann infer.Annotator) {
	ann.Describe(&a.InstanceName, "The name of the Multipass VM instance.")
	ann.Describe(&a.SourcePath, "The host directory path to mount into the VM.")
	ann.Describe(&a.TargetPath, "The target directory path inside the VM instance.")
	ann.Describe(&a.MountType, "The mount strategy to use (e.g., 'native' or 'classic'). Defaults to 'native'.")
	ann.SetDefault(&a.MountType, "native")
}

// Create mounts a directory into a VM.
func (m *Mount) Create(
	ctx context.Context, req infer.CreateRequest[MountArgs],
) (infer.CreateResponse[MountState], error) {
	inputs := req.Inputs
	id := inputs.InstanceName + ":" + inputs.TargetPath
	if req.DryRun {
		return infer.CreateResponse[MountState]{
			ID:     id,
			Output: MountState{MountArgs: inputs},
		}, nil
	}

	cfg := infer.GetConfig[Config](ctx)
	client := multipass.NewClient(cfg.MultipassBin)

	mountType := inputs.MountType
	if mountType == "" {
		mountType = "native"
	}

	if err := client.Mount(ctx, multipass.MountArgs{
		InstanceName: inputs.InstanceName,
		SourcePath:   inputs.SourcePath,
		TargetPath:   inputs.TargetPath,
		MountType:    mountType,
	}); err != nil {
		return infer.CreateResponse[MountState]{}, fmt.Errorf("mounting %s into %s:%s: %w",
			inputs.SourcePath, inputs.InstanceName, inputs.TargetPath, err)
	}

	return infer.CreateResponse[MountState]{ID: id, Output: MountState{MountArgs: inputs}}, nil
}

// Read checks whether the mount still exists.
func (m *Mount) Read(
	ctx context.Context, req infer.ReadRequest[MountArgs, MountState],
) (infer.ReadResponse[MountArgs, MountState], error) {
	inputs := req.Inputs
	cfg := infer.GetConfig[Config](ctx)
	client := multipass.NewClient(cfg.MultipassBin)

	info, err := client.Info(ctx, inputs.InstanceName)
	if err != nil {
		return infer.ReadResponse[MountArgs, MountState]{}, fmt.Errorf("reading mounts for %s: %w", inputs.InstanceName, err)
	}
	if info == nil {
		return infer.ReadResponse[MountArgs, MountState]{}, nil
	}

	if _, ok := info.Mounts[inputs.TargetPath]; ok {
		return infer.ReadResponse[MountArgs, MountState]{
			ID:     req.ID,
			Inputs: inputs,
			State:  MountState{MountArgs: inputs},
		}, nil
	}

	// Mount is absent.
	return infer.ReadResponse[MountArgs, MountState]{}, nil
}

// Delete unmounts the directory from the VM.
func (m *Mount) Delete(ctx context.Context, req infer.DeleteRequest[MountState]) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[Config](ctx)
	client := multipass.NewClient(cfg.MultipassBin)

	if err := client.Umount(ctx, req.State.InstanceName, req.State.TargetPath); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("unmounting %s:%s: %w", req.State.InstanceName, req.State.TargetPath, err)
	}
	return infer.DeleteResponse{}, nil
}

// Diff marks all input fields as requiring replacement.
func (m *Mount) Diff(
	ctx context.Context, req infer.DiffRequest[MountArgs, MountState],
) (infer.DiffResponse, error) {
	olds := req.State
	news := req.Inputs

	diff := map[string]p.PropertyDiff{}
	replaceFields := map[string]bool{
		"instanceName": olds.InstanceName != news.InstanceName,
		"sourcePath":   olds.SourcePath != news.SourcePath,
		"targetPath":   olds.TargetPath != news.TargetPath,
		"mountType":    olds.MountType != news.MountType,
	}

	hasChanges := false
	for field, changed := range replaceFields {
		if changed {
			diff[field] = p.PropertyDiff{Kind: p.UpdateReplace}
			hasChanges = true
		}
	}

	return infer.DiffResponse{
		HasChanges:   hasChanges,
		DetailedDiff: diff,
	}, nil
}
