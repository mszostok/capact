package metadata

import (
	"context"
	"fmt"

	"capact.io/capact/internal/ptr"
	"capact.io/capact/internal/regexutil"
	hubpublicgraphql "capact.io/capact/pkg/hub/api/graphql/public"
	"capact.io/capact/pkg/hub/client/public"

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
	ListTypes(ctx context.Context, opts ...public.TypeOption) ([]*hubpublicgraphql.Type, error)
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

	resolvedTypeRefs, err := r.hubCli.FindTypeInstancesTypeRef(ctx, idsToQuery)
	if err != nil {
		return errors.Wrap(err, "while finding TypeRef for TypeInstances")
	}

	// verify if all TypeInstances are resolved
	multiErr := multierror.New()
	for _, ti := range unresolvedTIs {
		if typeRef, exists := resolvedTypeRefs[ti.ID]; exists && typeRef.Path != "" && typeRef.Revision != "" {
			continue
		}

		multiErr = multierr.Append(multiErr, fmt.Errorf("missing Type reference for %s", ti.String(true)))
	}
	if multiErr.ErrorOrNil() != nil {
		return multiErr
	}

	typeRefWithParentNodes, err := r.enrichWithParentNodes(ctx, resolvedTypeRefs)
	if err != nil {
		return errors.Wrap(err, "while resolving parent nodes for TypeRefs")
	}

	r.setTypeRefsForAdditionalTypeInstances(policy, typeRefWithParentNodes)
	r.setTypeRefsForRequiredTypeInstances(policy, typeRefWithParentNodes)
	r.setTypeRefsForBackendTypeInstances(policy, typeRefWithParentNodes)

	return nil
}

type TypeRefWithParentNodes struct {
	types.TypeRef
	AdditionalRefs []string
}

func (r *Resolver) enrichWithParentNodes(ctx context.Context, refs map[string]hublocalgraphql.TypeInstanceTypeReference) (map[string]TypeRefWithParentNodes, error) {
	out := map[string]TypeRefWithParentNodes{}

	// TODO: unify...
	typesPath := r.mapToPathSlice(refs)
	const typeListQueryFields = public.TypeRevisionRootFields | public.TypeRevisionSpecAdditionalRefsField
	filter := regexutil.OrStringSlice(typesPath)
	res, err := r.hubCli.ListTypes(ctx, public.WithTypeRevisions(typeListQueryFields), public.WithTypeFilter(hubpublicgraphql.TypeFilter{
		PathPattern: ptr.String(filter),
	}))
	if err != nil {
		return nil, err
	}

	gotAttachedTypes := map[string][]string{}
	for _, item := range res {
		if item == nil {
			continue
		}
		for _, rev := range item.Revisions {
			if rev.Spec == nil {
				continue
			}
			gotAttachedTypes[r.key(item.Path, rev.Revision)] = rev.Spec.AdditionalRefs
		}
	}

	for id, ref := range refs {
		parents := gotAttachedTypes[r.key(ref.Path, ref.Revision)]
		out[id] = TypeRefWithParentNodes{
			TypeRef:        types.TypeRef(ref),
			AdditionalRefs: parents,
		}
	}

	return out, nil
}

func (r *Resolver) key(a, b string) string {
	return fmt.Sprintf("%s:%s", a, b)
}

func (r *Resolver) mapToPathSlice(in map[string]hublocalgraphql.TypeInstanceTypeReference) []string {
	var paths []string
	for _, expType := range in {
		paths = append(paths, expType.Path)
	}
	return paths
}

func (r *Resolver) setTypeRefsForRequiredTypeInstances(policy *policy.Policy, typeRefs map[string]TypeRefWithParentNodes) {
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
				policy.Interface.Rules[ruleIdx].OneOf[ruleItemIdx].Inject.RequiredTypeInstances[reqTIIdx].ExtendsHubBackend = r.isExtendingHubStorage(typeRef.AdditionalRefs)
			}
		}
	}
}

func (r *Resolver) setTypeRefsForAdditionalTypeInstances(policy *policy.Policy, typeRefs map[string]TypeRefWithParentNodes) {
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

func (r *Resolver) setTypeRefsForBackendTypeInstances(policy *policy.Policy, typeRefs map[string]TypeRefWithParentNodes) {
	for ruleIdx, rule := range policy.TypeInstance.Rules {
		typeRef, exists := typeRefs[rule.Backend.ID]
		if !exists {
			continue
		}

		policy.TypeInstance.Rules[ruleIdx].Backend.TypeRef = &types.ManifestRef{
			Path:     typeRef.Path,
			Revision: typeRef.Revision,
		}
		policy.TypeInstance.Rules[ruleIdx].Backend.ExtendsHubBackend = r.isExtendingHubStorage(typeRef.AdditionalRefs)
	}
}

func (r *Resolver) isExtendingHubStorage(parents []string) bool {
	for _, ref := range parents {
		if ref == types.HubBackendParentNodeName {
			return true
		}
	}
	return false
}
