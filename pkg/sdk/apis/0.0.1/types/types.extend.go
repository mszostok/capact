// Package types holds manually added types.
package types

import (
	"fmt"
	"strings"
)

const (
	// OCFPathPrefix defines path prefix that all OCF manifest must have.
	OCFPathPrefix = "cap."
	// HubBackendParentNodeName define parent path for the core hub storage.
	HubBackendParentNodeName = "cap.core.type.hub.storage"

	// maxBackendLookupForTypeRef defines maximum number of iteration to find a matching backend based on TypeRef path pattern.
	maxBackendLookupForTypeRef = 30
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

// TypeInstanceBackendCollection knows which Backend should be used for a given TypeInstance based on the TypeRef
type TypeInstanceBackendCollection struct {
	byTypeRef      map[string]TypeInstanceBackend
	byAlias        map[string]TypeInstanceBackend
	defaultBackend TypeInstanceBackend
}

type TypeInstanceBackend struct {
	ID          string
	Description *string
}

func (t *TypeInstanceBackendCollection) SetDefault(backend TypeInstanceBackend) {
	t.defaultBackend = backend
}

func (t *TypeInstanceBackendCollection) SetByTypeRef(typeRef TypeRef, backend TypeInstanceBackend) {
	if t.byTypeRef == nil {
		t.byTypeRef = map[string]TypeInstanceBackend{}
	}
	t.byTypeRef[t.key(typeRef)] = backend
}

// GetByTypeRef returns storage backend for a given TypeRef.
// If backend for an explicit TypeRef is not found, the pattern matching is used.
//
// For example, if TypeRef.path is `cap.type.capactio.examples.message`:
//    - cap.type.capactio.examples.*
//    - cap.type.capactio.*
//    - cap.type.*
//    - cap.*
//
// If both methods fail, default backend is returned.
func (t TypeInstanceBackendCollection) GetByTypeRef(typeRef TypeRef) TypeInstanceBackend {
	// 1. Try the explicit TypeRef
	backend, found := t.byTypeRef[t.key(typeRef)]
	if found {
		return backend
	}

	// 2. Try to find matching pattern for a given TypeRef.

	var (
		subPath    = typeRef.Path
		iterations = 0
	)

	for {
		if fmt.Sprintf("%s.", subPath) == OCFPathPrefix || iterations > maxBackendLookupForTypeRef {
			break
		}
		subPath = TrimLastNodeFromOCFPath(subPath)

		keyPatterns := []string{
			fmt.Sprintf("%s.*:%s", subPath, typeRef.Revision), // matching with revision has higher priority
			fmt.Sprintf("%s.*", subPath),                      // check for path pattern only
		}
		for _, pattern := range keyPatterns {
			backend, found := t.byTypeRef[pattern]
			if found {
				return backend
			}

		}
		iterations++
	}

	return t.defaultBackend
}

func (t *TypeInstanceBackendCollection) GetByAlias(name string) (TypeInstanceBackend, bool) {
	backend, found := t.byAlias[name]
	return backend, found
}

func (t *TypeInstanceBackendCollection) SetByAlias(name string, backend TypeInstanceBackend) {
	if t.byAlias == nil {
		t.byAlias = map[string]TypeInstanceBackend{}
	}
	t.byAlias[name] = backend
}

func (t TypeInstanceBackendCollection) key(typeRef TypeRef) string {
	if typeRef.Revision != "" {
		return fmt.Sprintf("%s:%s", typeRef.Path, typeRef.Revision)
	}
	return typeRef.Path
}

func (t *TypeInstanceBackendCollection) GetAll() map[string]TypeInstanceBackend {
	out := map[string]TypeInstanceBackend{}
	for k, v := range t.byAlias {
		out[k] = v
	}
	for k, v := range t.byTypeRef {
		out[k] = v
	}
	return out
}

func TrimLastNodeFromOCFPath(in string) string {
	idx := strings.LastIndex(in, ".")
	if idx == -1 {
		return in
	}

	return in[:idx]
}
