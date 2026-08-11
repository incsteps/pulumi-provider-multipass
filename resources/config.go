package resources

import (
	"github.com/pulumi/pulumi-go-provider/infer"
)

// Config holds provider-level configuration for the Multipass provider.
type Config struct {
	// MultipassBin is the path to the multipass binary. Defaults to "multipass" in PATH.
	MultipassBin string `pulumi:"multipassBin,optional"`
	// LaunchTimeout is the number of seconds to wait for an instance to start. Default 300.
	LaunchTimeout int `pulumi:"launchTimeout,optional"`
	// OperationTimeout is the number of seconds to wait for general operations. Default 60.
	OperationTimeout int `pulumi:"operationTimeout,optional"`
}

var _ infer.Annotated = (*Config)(nil)

func (c *Config) Annotate(a infer.Annotator) {
	a.Describe(&c.MultipassBin, "Path to the multipass binary. Defaults to 'multipass' in PATH.")
	a.SetDefault(&c.MultipassBin, "")
	a.Describe(&c.LaunchTimeout, "Seconds to wait for an instance to become Running. Default 300.")
	a.SetDefault(&c.LaunchTimeout, 300)
	a.Describe(&c.OperationTimeout, "Seconds to wait for general CLI operations. Default 60.")
	a.SetDefault(&c.OperationTimeout, 60)
}
