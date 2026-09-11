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
	ann.Describe(&a.Name, "The unique name of the Multipass virtual machine instance.")
	ann.Describe(&a.Image, "The OS image to launch (e.g., '24.04', 'daily:24.04'). Defaults to '24.04'.")
	ann.Describe(&a.Cpus, "The number of CPUs to allocate to the instance. Defaults to 1.")
	ann.Describe(&a.Memory, "The amount of RAM to allocate (e.g., '1G', '2048M'). Defaults to '1G'.")
	ann.Describe(&a.Disk, "The disk size to allocate (e.g., '5G', '10G'). Defaults to '5G'.")
	ann.Describe(&a.Cloudinit, "Path to a cloud-init user-data file or inline cloud-init configuration.")

	ann.SetDefault(&a.Image, "24.04")
	ann.SetDefault(&a.Cpus, 1)
	ann.SetDefault(&a.Memory, "1G")
	ann.SetDefault(&a.Disk, "5G")
}

func (a InstanceArgs) applyDefaults() InstanceArgs {
	if a.Image == "" {
		a.Image = "24.04"
	}
	if a.Cpus == 0 {
		a.Cpus = 1
	}
	if a.Memory == "" {
		a.Memory = "1G"
	}
	if a.Disk == "" {
		a.Disk = "5G"
	}
	return a
}

// Create launches a new Multipass instance.
func (i *Instance) Create(
	ctx context.Context, req infer.CreateRequest[InstanceArgs],
) (infer.CreateResponse[InstanceState], error) {

	inputs := req.Inputs.applyDefaults()

	if req.DryRun {
		return infer.CreateResponse[InstanceState]{
			ID:     inputs.Name,
			Output: InstanceState{InstanceArgs: inputs},
		}, nil
	}

	cfg := infer.GetConfig[Config](ctx)
	client := multipass.NewClient(cfg.MultipassBin)

	info, err := client.Launch(ctx, multipass.LaunchArgs{
		Name:      inputs.Name,
		Image:     inputs.Image,
		Cpus:      inputs.Cpus,
		Memory:    inputs.Memory,
		Disk:      inputs.Disk,
		Cloudinit: inputs.Cloudinit,
		Timeout:   cfg.LaunchTimeout,
	})
	if err != nil {
		// Idempotent create: if the instance already exists (e.g. due to a
		// provider binary upgrade causing ~provider diff + replace), adopt it.
		if strings.Contains(err.Error(), "already exists") {
			existing, readErr := client.Info(ctx, inputs.Name)
			if readErr == nil && existing != nil {
				state := instanceInfoToState(inputs, existing)
				return infer.CreateResponse[InstanceState]{ID: inputs.Name, Output: state}, nil
			}
		}
		return infer.CreateResponse[InstanceState]{}, fmt.Errorf("launching instance %s: %w", inputs.Name, err)
	}

	state := instanceInfoToState(inputs, info)
	return infer.CreateResponse[InstanceState]{ID: inputs.Name, Output: state}, nil
}

// Read refreshes the state of an existing Multipass instance.
func (i *Instance) Read(
	ctx context.Context, req infer.ReadRequest[InstanceArgs, InstanceState],
) (infer.ReadResponse[InstanceArgs, InstanceState], error) {
	cfg := infer.GetConfig[Config](ctx)
	client := multipass.NewClient(cfg.MultipassBin)

	info, err := client.Info(ctx, req.ID)
	if err != nil {
		return infer.ReadResponse[InstanceArgs, InstanceState]{}, fmt.Errorf("reading instance %s: %w", req.ID, err)
	}
	if info == nil {
		// Signal not-found by returning an empty response (empty ID).
		return infer.ReadResponse[InstanceArgs, InstanceState]{}, nil
	}

	newState := instanceInfoToState(req.Inputs, info)
	return infer.ReadResponse[InstanceArgs, InstanceState]{
		ID:     req.ID,
		Inputs: req.Inputs,
		State:  newState,
	}, nil
}

// Delete removes an instance.
func (i *Instance) Delete(ctx context.Context, req infer.DeleteRequest[InstanceState]) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[Config](ctx)
	client := multipass.NewClient(cfg.MultipassBin)

	if err := client.Delete(ctx, req.ID); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("deleting instance %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, client.Purge(ctx)
}

// Diff marks all input fields as requiring replacement.
func (i *Instance) Diff(
	ctx context.Context, req infer.DiffRequest[InstanceArgs, InstanceState],
) (infer.DiffResponse, error) {

	olds := req.State
	news := req.Inputs.applyDefaults()

	diff := map[string]p.PropertyDiff{}
	replaceFields := map[string]bool{
		"name":      olds.Name != news.Name,
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
	return infer.DiffResponse{
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
