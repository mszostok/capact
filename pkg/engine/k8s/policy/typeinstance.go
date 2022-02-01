package policy

import "capact.io/capact/pkg/sdk/apis/0.0.1/types"

// TypeInstancePolicy holds the Policy for TypeInstance.
type TypeInstancePolicy struct {
	Rules []RulesForTypeInstance `json:"rules"`
}

// RulesForTypeInstance holds a single policy rule for a TypeInstance.
// +kubebuilder:object:generate=true
type RulesForTypeInstance struct {
	TypeRef types.ManifestRefWithOptRevision `json:"typeRef"`
	Backend TypeInstanceBackend              `json:"backend"`
}

// TypeInstanceBackend holds a Backend description to be used for storing a given TypeInstance.
// +kubebuilder:object:generate=true
type TypeInstanceBackend struct {
	TypeInstanceReference `json:",inline"`
}
