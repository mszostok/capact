package client

import (
	"context"
	"net/http"

	hubpublicgraphql "capact.io/capact/pkg/hub/api/graphql/public"

	hublocalgraphql "capact.io/capact/pkg/hub/api/graphql/local"
	"capact.io/capact/pkg/hub/client/local"
	"capact.io/capact/pkg/hub/client/public"
	"github.com/machinebox/graphql"
)

// Client used to communicate with the Capact Hub GraphQL APIs
type Client struct {
	Local
	Public
}

// Local interface aggregates methods to interact with Capact Local Hub.
type Local interface {
	CreateTypeInstance(ctx context.Context, in *hublocalgraphql.CreateTypeInstanceInput) (string, error)
	CreateTypeInstances(ctx context.Context, in *hublocalgraphql.CreateTypeInstancesInput) ([]hublocalgraphql.CreateTypeInstanceOutput, error)
	FindTypeInstance(ctx context.Context, id string, opts ...local.TypeInstancesOption) (*hublocalgraphql.TypeInstance, error)
	ListTypeInstances(ctx context.Context, filter *hublocalgraphql.TypeInstanceFilter, opts ...local.TypeInstancesOption) ([]hublocalgraphql.TypeInstance, error)
	ListTypeInstancesTypeRef(ctx context.Context) ([]hublocalgraphql.TypeInstanceTypeReference, error)
	DeleteTypeInstance(ctx context.Context, id string) error
	LockTypeInstances(ctx context.Context, in *hublocalgraphql.LockTypeInstancesInput) error
	UnlockTypeInstances(ctx context.Context, in *hublocalgraphql.UnlockTypeInstancesInput) error
	UpdateTypeInstances(ctx context.Context, in []hublocalgraphql.UpdateTypeInstancesInput, opts ...local.TypeInstancesOption) ([]hublocalgraphql.TypeInstance, error)
	FindTypeInstancesTypeRef(ctx context.Context, ids []string) (map[string]hublocalgraphql.TypeInstanceTypeReference, error)
}

// Public interface aggregates methods to interact with Capact Public Hub.
type Public interface {
	ListTypes(ctx context.Context, opts ...public.TypeOption) ([]*hubpublicgraphql.Type, error)
	GetInterfaceLatestRevisionString(ctx context.Context, ref hubpublicgraphql.InterfaceReference) (string, error)
	FindInterfaceRevision(ctx context.Context, ref hubpublicgraphql.InterfaceReference, opts ...public.InterfaceRevisionOption) (*hubpublicgraphql.InterfaceRevision, error)
	ListImplementationRevisionsForInterface(ctx context.Context, ref hubpublicgraphql.InterfaceReference, opts ...public.ListImplementationRevisionsForInterfaceOption) ([]hubpublicgraphql.ImplementationRevision, error)
	ListInterfaces(ctx context.Context, opts ...public.InterfaceOption) ([]*hubpublicgraphql.Interface, error)
	ListImplementationRevisions(ctx context.Context, opts ...public.ListImplementationRevisionsOption) ([]*hubpublicgraphql.ImplementationRevision, error)
	CheckManifestRevisionsExist(ctx context.Context, manifestRefs []hubpublicgraphql.ManifestReference) (map[hubpublicgraphql.ManifestReference]bool, error)
}

// New returns a new Client to interact with the Capact Local and Public Hub.
func New(endpoint string, httpClient *http.Client) *Client {
	clientOpt := graphql.WithHTTPClient(httpClient)
	client := graphql.NewClient(endpoint, clientOpt)

	return &Client{
		Local:  local.NewClient(client),
		Public: public.NewClient(client),
	}
}
