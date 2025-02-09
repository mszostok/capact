// Package types holds manually added types.
package types

import (
	"fmt"
	"strings"
)

const (
	// OCFPathPrefix defines path prefix that all OCF manifest must have.
	OCFPathPrefix = "cap."
	// HubBackendParentNodeName defines parent path for the core Hub storage.
	HubBackendParentNodeName = "cap.core.type.hub.storage"
	// BuiltinHubStorageTypePath defines Type path for built-in Hub storage.
	BuiltinHubStorageTypePath = "cap.core.type.hub.storage.neo4j"
)

// InterfaceRef holds the full path and revision to the Interface
type InterfaceRef ManifestRefWithOptRevision

// ImplementationRef holds the full path and revision to the Implementation
type ImplementationRef ManifestRefWithOptRevision

// AttributeRef holds the full path and revision to the Attribute
type AttributeRef ManifestRefWithOptRevision

// ManifestRef holds the full path and the revision to a given manifest.
// +kubebuilder:object:generate=true
type ManifestRef struct {
	Path     string `json:"path"`     // Path of a given manifest
	Revision string `json:"revision"` // Version of the manifest content in the SemVer format.
}

func (in *ManifestRef) String() string {
	return fmt.Sprintf("%s:%s", in.Path, in.Revision)
}

// ManifestRefWithOptRevision specifies type by path and optional revision.
// +kubebuilder:object:generate=true
type ManifestRefWithOptRevision struct {
	// Path of a given Type.
	Path string `json:"path"`
	// Version of the manifest content in the SemVer format.
	Revision *string `json:"revision"`
}

func (in *ManifestRefWithOptRevision) String() string {
	if in == nil {
		return ""
	}
	out := in.Path
	if in.Revision != nil && *in.Revision != "" {
		out = fmt.Sprintf("%s:%s", out, *in.Revision)
	}

	return out
}

// InputTypeInstanceRef holds input TypeInstance reference.
type InputTypeInstanceRef struct {
	// Name refers to input TypeInstance name used in rendered Action.
	// Name is not unique as there may be multiple TypeInstances with the same name on different levels of Action workflow.
	Name string `json:"name"`

	// ID is a unique identifier for the input TypeInstance.
	ID string `json:"id"`
}

// ParametersCollection holds input parameters collection indexed by name.
type ParametersCollection map[string]string

// ManifestKind specifies OCF manifest kind.
type ManifestKind string

const (
	// RepoMetadataManifestKind specifies RepoMetadata kind.
	RepoMetadataManifestKind ManifestKind = "RepoMetadata"
	// TypeManifestKind specifies Type kind.
	TypeManifestKind ManifestKind = "Type"
	// AttributeManifestKind specifies Attribute kind.
	AttributeManifestKind ManifestKind = "Attribute"
	// InterfaceManifestKind specifies Interface kind.
	InterfaceManifestKind ManifestKind = "Interface"
	// ImplementationManifestKind specifies Implementation kind.
	ImplementationManifestKind ManifestKind = "Implementation"
	// InterfaceGroupManifestKind specifies InterfaceGroup kind.
	InterfaceGroupManifestKind ManifestKind = "InterfaceGroup"
	// VendorManifestKind specifies Vendor kind.
	VendorManifestKind ManifestKind = "Vendor"
)

// OCFVersion specifies the OCF version.
type OCFVersion string

// ManifestMetadata specifies the essential, common OCF manifest metadata.
type ManifestMetadata struct {
	OCFVersion OCFVersion   `yaml:"ocfVersion"`
	Kind       ManifestKind `yaml:"kind"`
}

// TrimLastNodeFromOCFPath removes the last node name from the given OCF path.
// For example, for `cap.core.type.examples.name` returns `cap.core.type.examples`.
func TrimLastNodeFromOCFPath(in string) string {
	idx := strings.LastIndex(in, ".")
	if idx == -1 {
		return in
	}

	return in[:idx]
}
