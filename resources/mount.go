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

func (m *Mount) Annotate(a infer.Annotator) {
	a.Describe(m, "A host directory mount into a Multipass virtual machine instance.")
}

func (a *MountArgs) Annotate(ann infer.Annotator) {
	ann.SetDefault(&a.MountType, "native")
}

// Create mounts a directory into a VM.
func (m *Mount) Create(
	ctx context.Context, name string, inputs MountArgs, preview bool,
) (string, MountState, error) {
	id := inputs.InstanceName + ":" + inputs.TargetPath
	if preview {
		return id, MountState{MountArgs: inputs}, nil
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
		return "", MountState{}, fmt.Errorf("mounting %s into %s:%s: %w",
			inputs.SourcePath, inputs.InstanceName, inputs.TargetPath, err)
	}

	return id, MountState{MountArgs: inputs}, nil
}

// Read checks whether the mount still exists.
func (m *Mount) Read(
	ctx context.Context, id string, inputs MountArgs, state MountState,
) (string, MountArgs, MountState, error) {
	cfg := infer.GetConfig[Config](ctx)
	client := multipass.NewClient(cfg.MultipassBin)

	info, err := client.Info(ctx, inputs.InstanceName)
	if err != nil {
		return "", inputs, state, fmt.Errorf("reading mounts for %s: %w", inputs.InstanceName, err)
	}
	if info == nil {
		return "", inputs, MountState{}, nil
	}

	if _, ok := info.Mounts[inputs.TargetPath]; ok {
		return id, inputs, MountState{MountArgs: inputs}, nil
	}

	// Mount is absent.
	return "", inputs, MountState{}, nil
}

// Delete unmounts the directory from the VM.
func (m *Mount) Delete(ctx context.Context, id string, props MountState) error {
	cfg := infer.GetConfig[Config](ctx)
	client := multipass.NewClient(cfg.MultipassBin)

	if err := client.Umount(ctx, props.InstanceName, props.TargetPath); err != nil {
		return fmt.Errorf("unmounting %s:%s: %w", props.InstanceName, props.TargetPath, err)
	}
	return nil
}

// Diff marks all input fields as requiring replacement.
func (m *Mount) Diff(
	ctx context.Context, id string, olds MountState, news MountArgs,
) (p.DiffResponse, error) {
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

	return p.DiffResponse{
		HasChanges:   hasChanges,
		DetailedDiff: diff,
	}, nil
}
