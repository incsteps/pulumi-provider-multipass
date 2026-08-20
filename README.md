# pulumi-provider-multipass

A [Pulumi native provider](https://www.pulumi.com/docs/using-pulumi/pulumi-packages/authoring/) for
[Canonical Multipass](https://multipass.run) — declarative, snapshot-aware VM management via the
`multipass` CLI.

Multipass is the fastest way to get reproducible Ubuntu VMs on a laptop or workstation. This provider
makes those VMs a first-class Pulumi resource, so a multi-node topology can be described in code,
diffed, snapshotted, and torn down without cloud spend.

---

## Resources

| Resource | Description |
|---|---|
| `multipass.Instance` | Provision a VM (name, image, CPUs, memory, disk, cloud-init) |
| `multipass.Snapshot` | Capture a named snapshot of a VM |
| `multipass.Mount` | Mount a host directory into a running VM |

## Functions

| Function | Description |
|---|---|
| `multipass.restore` | Restore a VM from a named snapshot (via the Pulumi Automation API) |

## Provider config

| Key | Default | Description |
|---|---|---|
| `multipass:binaryPath` | `multipass` | Path to the `multipass` CLI binary |

---

## Install

The provider is distributed as GitHub release assets rather than through the public Pulumi registry.
`pluginDownloadURL` is baked into the schema, so the CLI knows where to look:

```bash
pulumi plugin install resource multipass v0.1.0 \
  --server github://api.github.com/incsteps/pulumi-provider-multipass
```

Then add the SDK to your project:

```bash
pulumi package add multipass
```

### Usage

```typescript
import * as multipass from "@incsteps/pulumi-multipass";

const vm = new multipass.resources.Instance("dev", {
    name:   "dev",
    image:  "24.04",
    cpus:   2,
    memory: "4G",
    disk:   "20G",
});

export const ip = vm.ipv4;
```

---

## Build from source

Requires **Go 1.24+**, the **Pulumi CLI**, and **Multipass** installed on macOS or Linux.

```bash
make build      # compile bin/pulumi-resource-multipass
make install    # install into ~/.pulumi/plugins/resource-multipass-v0.1.0/
make test       # unit tests (mocked CLI — no VMs created)
```

Regenerating the schema and SDKs after changing resource definitions:

```bash
make schema     # extract schema.json from the running provider over gRPC
make gen-sdk    # regenerate sdk/nodejs and sdk/go from that schema
```

Both are derived artifacts — edit `provider/`, `resources/`, and `functions/`, never `sdk/` by hand.

> Package naming (npm scope, Go import path, plugin download URL) is set in
> `provider/provider.go` via `WithNamespace` / `WithLanguageMap`, and flows into the generated SDKs.
> Changing it there and re-running `make gen-sdk` is the only supported way to rename the packages.

### Integration tests

`make test-integration` drives **real Multipass VMs** — it is slow and requires Multipass to be
installed and running. Unit tests mock the CLI boundary and are safe to run anywhere.

---

## Releasing

Tag a version; CI cross-compiles the plugin for linux/darwin/windows on amd64+arm64, attaches the
tarballs to the GitHub release, and publishes the nodejs SDK to npm:

```bash
git tag v0.1.0 && git push origin v0.1.0
```

npm publishing is skipped until an `NPM_TOKEN` secret is configured on the repository; the plugin
tarballs are attached regardless.

---

## License

Apache-2.0.
