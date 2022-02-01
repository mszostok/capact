package metadata

import (
	"context"
	"fmt"

	"capact.io/capact/internal/multierror"

	"capact.io/capact/pkg/engine/k8s/policy"
	hublocalgraphql "capact.io/capact/pkg/hub/api/graphql/local"
	"capact.io/capact/pkg/sdk/apis/0.0.1/types"
	multierr "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

// HubClient defines Hub client which is able to find TypeInstance Type references.
type HubClient interface {
	FindTypeInstancesTypeRef(ctx context.Context, ids []string) (map[string]hublocalgraphql.TypeInstanceTypeReference, error)
}

// Resolver resolves Policy metadata against Hub.
type Resolver struct {
	hubCli HubClient
}

// NewResolver returns new Resolver instance.
func NewResolver(hubCli HubClient) *Resolver {
	return &Resolver{hubCli: hubCli}
}

// ResolveTypeInstanceMetadata resolves needed TypeInstance metadata based on IDs for a given Policy.
func (r *Resolver) ResolveTypeInstanceMetadata(ctx context.Context, policy *policy.Policy) error {
	if policy == nil {
		return errors.New("policy cannot be nil")
	}

	if r.hubCli == nil {
		return errors.New("hub client cannot be nil")
	}

	unresolvedTIs := TypeInstanceIDsWithUnresolvedMetadataForPolicy(*policy)

	var idsToQuery []string
	for _, ti := range unresolvedTIs {
		idsToQuery = append(idsToQuery, ti.ID)
	}

	if len(idsToQuery) == 0 {
		return nil
	}

	res, err := r.hubCli.FindTypeInstancesTypeRef(ctx, idsToQuery)
	if err != nil {
		return errors.Wrap(err, "while finding TypeRef for TypeInstances")
	}

	// verify if all TypeInstances are resolved
	multiErr := multierror.New()
	for _, ti := range unresolvedTIs {
		if typeRef, exists := res[ti.ID]; exists && typeRef.Path != "" && typeRef.Revision != "" {
			continue
		}

		multiErr = multierr.Append(multiErr, fmt.Errorf("missing Type reference for %s", ti.String(true)))
	}
	if multiErr.ErrorOrNil() != nil {
		return multiErr
	}

	r.setTypeRefsForRequiredTypeInstances(policy, res)
	r.setTypeRefsForAdditionalTypeInstances(policy, res)
	r.setTypeRefsForBackendTypeInstances(policy, res) // probably change and verify that it's attached to `cap.core.type.hub.storage` node.

	return nil
}

func (r *Resolver) setTypeRefsForRequiredTypeInstances(policy *policy.Policy, typeRefs map[string]hublocalgraphql.TypeInstanceTypeReference) {
	for ruleIdx, rule := range policy.Interface.Rules {
		for ruleItemIdx, ruleItem := range rule.OneOf {
			if ruleItem.Inject == nil {
				continue
			}
			for reqTIIdx, reqTI := range ruleItem.Inject.RequiredTypeInstances {
				typeRef, exists := typeRefs[reqTI.ID]
				if !exists {
					continue
				}

				policy.Interface.Rules[ruleIdx].OneOf[ruleItemIdx].Inject.RequiredTypeInstances[reqTIIdx].TypeRef = &types.ManifestRef{
					Path:     typeRef.Path,
					Revision: typeRef.Revision,
				}
			}
		}
	}
}

func (r *Resolver) setTypeRefsForAdditionalTypeInstances(policy *policy.Policy, typeRefs map[string]hublocalgraphql.TypeInstanceTypeReference) {
	for ruleIdx, rule := range policy.Interface.Rules {
		for ruleItemIdx, ruleItem := range rule.OneOf {
			if ruleItem.Inject == nil {
				continue
			}
			for reqTIIdx, reqTI := range ruleItem.Inject.AdditionalTypeInstances {
				typeRef, exists := typeRefs[reqTI.ID]
				if !exists {
					continue
				}

				policy.Interface.Rules[ruleIdx].OneOf[ruleItemIdx].Inject.AdditionalTypeInstances[reqTIIdx].TypeRef = &types.ManifestRef{
					Path:     typeRef.Path,
					Revision: typeRef.Revision,
				}
			}
		}
	}
}

func (r *Resolver) setTypeRefsForBackendTypeInstances(policy *policy.Policy, typeRefs map[string]hublocalgraphql.TypeInstanceTypeReference) {
	for ruleIdx, rule := range policy.TypeInstance.Rules {
		typeRef, exists := typeRefs[rule.Backend.ID]
		if !exists {
			continue
		}

		policy.TypeInstance.Rules[ruleIdx].Backend.TypeRef = &types.ManifestRef{
			Path:     typeRef.Path,
			Revision: typeRef.Revision,
		}
	}
}
