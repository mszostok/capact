// Package types holds manually added types.
package types

import (
	"fmt"
	"path/filepath"
	"strings"
)

// OCFPathPrefix defines path prefix that all OCF manifest must have.
const OCFPathPrefix = "cap."

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

// ManifestRefWithOptRevision specifies type by path and optional revision.
// +kubebuilder:object:generate=true
type ManifestRefWithOptRevision struct {
	// Path of a given Type.
	Path string `json:"path"`
	// Version of the manifest content in the SemVer format.
	Revision *string `json:"revision"`
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

var DefaultTypeInstanceBackendKey = TypeRef{
	Path:     "default.typeinstance.backend.key",
	Revision: "default.typeinstance.backend.key",
}

// TypeInstanceBackendCollection knows which Backend should be used for a given TypeInstance based on the TypeRef
type TypeInstanceBackendCollection struct {
	data map[string]TypeInstanceBackend
}

type TypeInstanceBackend struct {
	ID          string
	Description *string
}

func (t *TypeInstanceBackendCollection) Set(typeRef TypeRef, backend TypeInstanceBackend) {
	if t.data == nil {
		t.data = map[string]TypeInstanceBackend{}
	}

	t.data[t.key(typeRef)] = backend
}

func (t TypeInstanceBackendCollection) Get(typeRef TypeRef) TypeInstanceBackend {
	// 1. Try the explict TypeRef
	backend, found := t.data[t.key(typeRef)]
	if found {
		return backend
	}

	// 2. Try to find matching entry for a given TypeRef.
	// For example, if type ref is `cap.type.capactio.examples.message`:
	//    - cap.type.capactio.examples.*
	//    - cap.type.capactio.*
	//    - cap.type.*
	//    - cap.*
	const (
		stopper           = "cap"
		maxIterationGuard = 30
	)

	iterations := 0
	for {
		if typeRef.Path == stopper || iterations > maxIterationGuard {
			break
		}

		typeRef.Path = strings.TrimSuffix(typeRef.Path, filepath.Ext(typeRef.Path))
		keyPattern := fmt.Sprintf("%s.*", typeRef.Path)
		fmt.Println(keyPattern)
		backend, found := t.data[keyPattern]
		if found {
			return backend
		}
		iterations++
	}

	return t.data[t.key(DefaultTypeInstanceBackendKey)]
}

func (t TypeInstanceBackendCollection) key(typeRef TypeRef) string {
	if typeRef.Revision != "" {
		return fmt.Sprintf("%s:%s", typeRef.Path, typeRef.Revision)
	}
	return typeRef.Path
}
