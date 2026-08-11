package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/incsteps/pulumi-provider-multipass/internal/multipass"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// InstanceArgs are the inputs for the Instance resource.
type InstanceArgs struct {
	Name      string `pulumi:"name"`
	Image     string `pulumi:"image,optional"`
	Cpus      int    `pulumi:"cpus,optional"`
	Memory    string `pulumi:"memory,optional"`
	Disk      string `pulumi:"disk,optional"`
	Cloudinit string `pulumi:"cloudinit,optional"`
}

// InstanceState is the full output state for the Instance resource.
type InstanceState struct {
	InstanceArgs
	IPv4         string   `pulumi:"ipv4"`
	AllIPv4      []string `pulumi:"allIpv4"`
	State        string   `pulumi:"state"`
	ImageHash    string   `pulumi:"imageHash"`
	ImageRelease string   `pulumi:"imageRelease"`
}

// Instance is the resource controller for Multipass VMs.
type Instance struct{}

var _ infer.CustomResource[InstanceArgs, InstanceState] = (*Instance)(nil)
var _ infer.CustomRead[InstanceArgs, InstanceState] = (*Instance)(nil)
var _ infer.CustomDelete[InstanceState] = (*Instance)(nil)
var _ infer.CustomDiff[InstanceArgs, InstanceState] = (*Instance)(nil)
var _ infer.Annotated = (*Instance)(nil)

func (i *Instance) Annotate(a infer.Annotator) {
	a.Describe(i, "A Multipass virtual machine instance.")
}

func (a *InstanceArgs) Annotate(ann infer.Annotator) {
	ann.SetDefault(&a.Image, "24.04")
	ann.SetDefault(&a.Cpus, 1)
	ann.SetDefault(&a.Memory, "1G")
	ann.SetDefault(&a.Disk, "5G")
}

// Create launches a new Multipass instance.
func (i *Instance) Create(
	ctx context.Context, name string, inputs InstanceArgs, preview bool,
) (string, InstanceState, error) {
	if preview {
		return inputs.Name, InstanceState{InstanceArgs: inputs}, nil
	}

	cfg := infer.GetConfig[Config](ctx)
	client := multipass.NewClient(cfg.MultipassBin)

	image := inputs.Image
	if image == "" {
		image = "24.04"
	}
	cpus := inputs.Cpus
	if cpus == 0 {
		cpus = 1
	}
	memory := inputs.Memory
	if memory == "" {
		memory = "1G"
	}
	disk := inputs.Disk
	if disk == "" {
		disk = "5G"
	}

	info, err := client.Launch(ctx, multipass.LaunchArgs{
		Name:      inputs.Name,
		Image:     image,
		Cpus:      cpus,
		Memory:    memory,
		Disk:      disk,
		Cloudinit: inputs.Cloudinit,
	})
	if err != nil {
		// Idempotent create: if the instance already exists (e.g. due to a
		// provider binary upgrade causing ~provider diff + replace), adopt it.
		if strings.Contains(err.Error(), "already exists") {
			existing, readErr := client.Info(ctx, inputs.Name)
			if readErr == nil && existing != nil {
				state := instanceInfoToState(inputs, existing)
				return inputs.Name, state, nil
			}
		}
		return "", InstanceState{}, fmt.Errorf("launching instance %s: %w", inputs.Name, err)
	}

	state := instanceInfoToState(inputs, info)
	return inputs.Name, state, nil
}

// Read refreshes the state of an existing Multipass instance.
func (i *Instance) Read(
	ctx context.Context, id string, inputs InstanceArgs, state InstanceState,
) (string, InstanceArgs, InstanceState, error) {
	cfg := infer.GetConfig[Config](ctx)
	client := multipass.NewClient(cfg.MultipassBin)

	info, err := client.Info(ctx, id)
	if err != nil {
		return "", inputs, state, fmt.Errorf("reading instance %s: %w", id, err)
	}
	if info == nil {
		// Signal not-found by returning empty ID.
		return "", inputs, InstanceState{}, nil
	}

	newState := instanceInfoToState(inputs, info)
	return id, inputs, newState, nil
}

// Delete removes an instance.
func (i *Instance) Delete(ctx context.Context, id string, props InstanceState) error {
	cfg := infer.GetConfig[Config](ctx)
	client := multipass.NewClient(cfg.MultipassBin)

	if err := client.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting instance %s: %w", id, err)
	}
	return client.Purge(ctx)
}

// Diff marks all input fields as requiring replacement.
func (i *Instance) Diff(
	ctx context.Context, id string, olds InstanceState, news InstanceArgs,
) (p.DiffResponse, error) {
	diff := map[string]p.PropertyDiff{}
	replaceFields := map[string]bool{
		"image":     olds.Image != news.Image,
		"cpus":      olds.Cpus != news.Cpus,
		"memory":    olds.Memory != news.Memory,
		"disk":      olds.Disk != news.Disk,
		"cloudinit": olds.Cloudinit != news.Cloudinit,
	}

	hasChanges := false
	for field, changed := range replaceFields {
		if changed {
			diff[field] = p.PropertyDiff{Kind: p.UpdateReplace}
			hasChanges = true
		}
	}

	// DeleteBeforeReplace: Multipass VM names are resource IDs — two VMs with the same
	// name cannot coexist. Without this flag, Pulumi's default create-before-delete
	// order calls Create with the existing name, which adopts the old VM, then deletes
	// the "old" resource — which is now the newly-adopted VM. Net result: VM vanishes.
	return p.DiffResponse{
		DeleteBeforeReplace: hasChanges,
		HasChanges:          hasChanges,
		DetailedDiff:        diff,
	}, nil
}

// instanceInfoToState converts multipass.InstanceInfo to InstanceState.
func instanceInfoToState(inputs InstanceArgs, info *multipass.InstanceInfo) InstanceState {
	state := InstanceState{InstanceArgs: inputs}
	if info == nil {
		return state
	}
	state.State = info.State
	state.AllIPv4 = info.IPv4
	if len(info.IPv4) > 0 {
		state.IPv4 = info.IPv4[0]
	}
	if info.Image != nil {
		state.ImageHash = info.Image.ID
		state.ImageRelease = info.Image.Release
	}
	return state
}
