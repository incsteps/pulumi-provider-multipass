package multipass

// InfoOutput is the top-level structure returned by `multipass info --format json`.
type InfoOutput struct {
	Info map[string]*InstanceInfo `json:"info"`
}

// InstanceInfo holds the per-instance fields from `multipass info --format json`.
type InstanceInfo struct {
	State   string            `json:"state"`
	IPv4    []string          `json:"ipv4"`
	Image   *ImageInfo        `json:"image"`
	Mounts  map[string]*Mount `json:"mounts"`
	// Snapshots is present when --snapshots flag is used.
	// The key is the snapshot name.
	Snapshots map[string]*SnapshotInfo `json:"snapshots"`
}

// ImageInfo holds image metadata.
type ImageInfo struct {
	ID      string `json:"id"`
	Release string `json:"release"`
}

// Mount holds a single mount entry from `multipass info`.
type Mount struct {
	SourcePath string        `json:"source_path"`
	UIDMappings []string     `json:"uid_mappings"`
	GIDMappings []string     `json:"gid_mappings"`
	MountType   string       `json:"mount_type"`
}

// SnapshotInfo holds a single snapshot entry from `multipass info --snapshots`.
// Name is populated by InfoSnapshots from the map key — it is not in the JSON body.
type SnapshotInfo struct {
	Name    string `json:"-"`
	Comment string `json:"comment"`
	Parent  string `json:"parent"`
	// Created is the human-readable timestamp from Multipass ("Fri May  8 ...").
	Created string `json:"created"`
	// Size is the disk delta as a human string (e.g. "1.5GiB"); may be empty.
	Size string `json:"size"`
}

// LaunchArgs holds the parameters for launching a new instance.
type LaunchArgs struct {
	Name      string
	Image     string
	Cpus      int
	Memory    string
	Disk      string
	Cloudinit string // inline YAML — written to a temp file, not a path
}

// SnapshotArgs holds the parameters for creating a snapshot.
type SnapshotArgs struct {
	InstanceName string
	SnapshotName string
	Comment      string
}

// MountArgs holds the parameters for mounting a directory into an instance.
type MountArgs struct {
	InstanceName string
	SourcePath   string
	TargetPath   string
	MountType    string // "classic" or "native"
}

// RestoreArgs holds the parameters for restoring a snapshot.
type RestoreArgs struct {
	InstanceName       string
	SnapshotName       string
	DestructiveRestore bool
}
