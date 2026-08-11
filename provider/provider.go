package provider

import (
	"github.com/incsteps/pulumi-provider-multipass/functions"
	"github.com/incsteps/pulumi-provider-multipass/resources"
	"github.com/pulumi/pulumi-go-provider/infer"
)

const (
	Name      = "multipass"
	Version   = "v0.1.0"
	Namespace = "incsteps"

	// Repository is the canonical source location. It also seeds the
	// PluginDownloadURL below, which is what lets `pulumi plugin install`
	// resolve the provider binary straight from a GitHub release.
	Repository = "https://github.com/incsteps/pulumi-provider-multipass"

	// pluginDownloadURL tells the Pulumi CLI to fetch release assets from
	// GitHub rather than the public Pulumi registry.
	pluginDownloadURL = "github://api.github.com/incsteps/pulumi-provider-multipass"
)

// Build returns the configured ProviderBuilder for the multipass provider.
func Build() *infer.ProviderBuilder {
	return infer.NewProviderBuilder().
		WithName(Name).
		WithVersion(Version).
		WithNamespace(Namespace).
		WithDisplayName("Multipass").
		WithDescription("A Pulumi native provider for Canonical Multipass — declarative, snapshot-aware VM management via the multipass CLI.").
		WithKeywords("pulumi", "multipass", "vm", "canonical", "category/infrastructure").
		WithLicense("Apache-2.0").
		WithHomepage(Repository).
		WithRepository(Repository).
		WithPluginDownloadURL(pluginDownloadURL).
		// Without an explicit language map the Go SDK generates against the
		// placeholder module path "example.com/..." (unbuildable), and the
		// nodejs SDK claims the upstream "@pulumi/" npm scope, which we do
		// not own. Pin both to paths this project actually controls.
		WithLanguageMap(map[string]any{
			"go": map[string]any{
				"importBasePath":                 Repository[len("https://"):] + "/sdk/go/multipass",
				"generateResourceContainerTypes": true,
				"respectSchemaVersion":           true,
			},
			"nodejs": map[string]any{
				"packageName":          "@incsteps/pulumi-multipass",
				"respectSchemaVersion": true,
			},
		}).
		WithConfig(infer.Config[resources.Config]()).
		WithResources(
			infer.Resource[*resources.Instance, resources.InstanceArgs, resources.InstanceState](),
			infer.Resource[*resources.Snapshot, resources.SnapshotArgs, resources.SnapshotState](),
			infer.Resource[*resources.Mount, resources.MountArgs, resources.MountState](),
		).
		WithFunctions(
			infer.Function[*functions.Restore, functions.RestoreArgs, functions.RestoreResult](),
		)
}
