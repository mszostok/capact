package action

import (
	"context"
	"errors"
	"testing"

	"capact.io/capact/pkg/sdk/validation"

	"capact.io/capact/pkg/sdk/apis/0.0.1/types"

	"capact.io/capact/internal/cli/heredoc"
	gqllocalapi "capact.io/capact/pkg/hub/api/graphql/local"
	gqlpublicapi "capact.io/capact/pkg/hub/api/graphql/public"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

var interfaceRevisionRaw = []byte(`
spec:
  input:
    parameters:
      - name: input-parameters
        jsonSchema: |-
          {
            "$schema": "http://json-schema.org/draft-07/schema",
            "type": "object",
            "required": [ "key" ],
            "properties": {
              "key": {
                "type": "boolean",
                "title": "Key"
              }
            }
          }
      - name: db-settings
        jsonSchema: |-
          {
            "$schema": "http://json-schema.org/draft-07/schema",
            "type": "object",
            "required": [ "key" ],
            "properties": {
              "key": {
                "type": "boolean",
                "title": "Key"
              }
            }
          }
      - name: aws-creds
        typeRef:
          path: cap.type.aws.auth.creds
          revision: 0.1.0`)

func TestValidateInterfaceInputParameters(t *testing.T) {
	// given
	iface := &gqlpublicapi.InterfaceRevision{}
	require.NoError(t, yaml.Unmarshal(interfaceRevisionRaw, iface))

	tests := map[string]struct {
		givenHubTypeInstances []*gqlpublicapi.TypeRevision
		givenParameters       types.ParametersCollection
		expectedIssues        string
	}{
		"Happy path JSON": {
			givenHubTypeInstances: []*gqlpublicapi.TypeRevision{
				fixAWSCredsTypeRev(),
			},
			givenParameters: types.ParametersCollection{
				"input-parameters": `{"key": true}`,
				"db-settings":      `{"key": true}`,
				"aws-creds":        `{"key": "true"}`,
			},
		},
		"Happy path YAML": {
			givenHubTypeInstances: []*gqlpublicapi.TypeRevision{
				fixAWSCredsTypeRev(),
			},
			givenParameters: types.ParametersCollection{
				"input-parameters": `key: true`,
				"db-settings":      `key: true`,
				"aws-creds":        `key: "true"`,
			},
		},
		"Not found `aws-creds`": {
			givenHubTypeInstances: []*gqlpublicapi.TypeRevision{
				fixAWSCredsTypeRev(),
			},
			givenParameters: types.ParametersCollection{
				"input-parameters": `{"key": true}`,
				"db-settings":      `{"key": true}`,
			},
			expectedIssues: heredoc.Doc(`
        	            	- Parameters "aws-creds":
        	            	    * required but missing input parameters`),
		},
		"Invalid parameters": {
			givenHubTypeInstances: []*gqlpublicapi.TypeRevision{
				fixAWSCredsTypeRev(),
			},
			givenParameters: types.ParametersCollection{
				"input-parameters": `{"key": "true"}`,
				"db-settings":      `{"key": "true"}`,
				"aws-creds":        `{"key": "true"}`,
			},
			expectedIssues: heredoc.Doc(`
        	            	- Parameters "db-settings":
        	            	    * key: Invalid type. Expected: boolean, given: string
        	            	- Parameters "input-parameters":
        	            	    * key: Invalid type. Expected: boolean, given: string`),
		},
	}
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			// given
			ctx := context.Background()
			fakeCli := &fakeHubCli{
				Types: tc.givenHubTypeInstances,
			}

			validator := NewValidator(fakeCli)

			// when
			ifaceSchemas, err := validator.LoadIfaceInputParametersSchemas(ctx, iface)
			// then
			require.NoError(t, err)
			require.Len(t, ifaceSchemas, 3)

			// when
			result, err := validator.ValidateParameters(ctx, ifaceSchemas, tc.givenParameters)
			// then
			require.NoError(t, err)

			if tc.expectedIssues == "" {
				assert.NoError(t, result.ErrorOrNil())
			} else {
				assert.EqualError(t, result.ErrorOrNil(), tc.expectedIssues)
			}
		})
	}
}

func TestValidateParametersNoop(t *testing.T) {
	tests := map[string]struct {
		givenIface      *gqlpublicapi.InterfaceRevision
		givenParameters types.ParametersCollection
	}{
		"Should do nothing on nil": {
			givenIface:      nil,
			givenParameters: nil,
		},
		"Should do nothing on zero values": {
			givenIface:      &gqlpublicapi.InterfaceRevision{},
			givenParameters: types.ParametersCollection{},
		},
	}
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			// given
			ctx := context.Background()

			validator := NewValidator(&fakeHubCli{})

			// when
			ifaceSchemas, err := validator.LoadIfaceInputParametersSchemas(ctx, tc.givenIface)
			// then
			require.NoError(t, err)
			require.Len(t, ifaceSchemas, 0)

			// when
			result, err := validator.ValidateParameters(ctx, ifaceSchemas, tc.givenParameters)
			// then
			require.NoError(t, err)
			assert.NoError(t, result.ErrorOrNil())
		})
	}
}

var InterfaceInputTypesRaw = []byte(`
spec:
  input:
    typeInstances:
      - name: database
        typeRef:
          path: cap.type.db.connection
          revision: 0.1.0
      - name: config
        typeRef:
          path: cap.type.mattermost.config
          revision: 0.1.0`)

func TestValidateTypeInstances(t *testing.T) {
	// given
	iface := &gqlpublicapi.InterfaceRevision{}
	require.NoError(t, yaml.Unmarshal(InterfaceInputTypesRaw, iface))

	tests := map[string]struct {
		givenHubTypeInstances map[string]gqllocalapi.TypeInstanceTypeReference
		givenTypeInstances    []types.InputTypeInstanceRef
		expectedIssues        string
	}{
		"Happy path": {
			givenHubTypeInstances: map[string]gqllocalapi.TypeInstanceTypeReference{
				"id-database": {
					Path:     "cap.type.db.connection",
					Revision: "0.1.0",
				},
				"id-config": {
					Path:     "cap.type.mattermost.config",
					Revision: "0.1.0",
				},
			},
			givenTypeInstances: []types.InputTypeInstanceRef{
				{Name: "config", ID: "id-config"},
				{Name: "database", ID: "id-database"},
			},
		},
		"Revision mismatch": {
			givenHubTypeInstances: map[string]gqllocalapi.TypeInstanceTypeReference{
				"id-database": {
					Path:     "cap.type.db.connection",
					Revision: "0.1.0",
				},
				"id-config": {
					Path:     "cap.type.mattermost.config",
					Revision: "0.1.1",
				},
			},
			givenTypeInstances: []types.InputTypeInstanceRef{
				{Name: "config", ID: "id-config"},
				{Name: "database", ID: "id-database"},
			},
			expectedIssues: heredoc.Doc(`
                    - TypeInstances "config":
                        * must be in Revision "0.1.0" but it's "0.1.1"`),
		},
		"Type mismatch": {
			givenHubTypeInstances: map[string]gqllocalapi.TypeInstanceTypeReference{
				"id-database": {
					Path:     "cap.type.db.connection",
					Revision: "0.1.0",
				},
				"id-config": {
					Path:     "cap.type.slack.config",
					Revision: "0.1.0",
				},
			},
			givenTypeInstances: []types.InputTypeInstanceRef{
				{Name: "config", ID: "id-config"},
				{Name: "database", ID: "id-database"},
			},
			expectedIssues: heredoc.Doc(`
                    - TypeInstances "config":
                        * must be of Type "cap.type.mattermost.config" but it's "cap.type.slack.config"`),
		},
		"not found required TypeInstance": {
			givenHubTypeInstances: map[string]gqllocalapi.TypeInstanceTypeReference{
				"id-database": {
					Path:     "cap.type.db.connection",
					Revision: "0.1.0",
				},
				"id-config": {
					Path:     "cap.type.mattermost.config",
					Revision: "0.1.0",
				},
			},
			givenTypeInstances: []types.InputTypeInstanceRef{
				{Name: "config", ID: "id-config"},
			},
			expectedIssues: heredoc.Doc(`
		           - TypeInstances "database":
		               * required but missing TypeInstance of type cap.type.db.connection:0.1.0`),
		},
	}
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			// given
			ctx := context.Background()
			fakeCli := &fakeHubCli{
				IDsTypeRefs: tc.givenHubTypeInstances,
			}

			validator := NewValidator(fakeCli)

			// when
			ifaceTypes, err := validator.LoadIfaceInputTypeInstanceRefs(ctx, iface)
			// then
			require.NoError(t, err)
			require.Len(t, ifaceTypes, 2)

			// when
			result, err := validator.ValidateTypeInstances(ctx, ifaceTypes, tc.givenTypeInstances)
			// then
			require.NoError(t, err)

			if tc.expectedIssues == "" {
				assert.NoError(t, result.ErrorOrNil())
			} else {
				assert.EqualError(t, result.ErrorOrNil(), tc.expectedIssues)
			}
		})
	}
}

func TestValidateTypeInstancesNoop(t *testing.T) {
	// given
	tests := map[string]struct {
		givenIface         *gqlpublicapi.InterfaceRevision
		givenTypeInstances []types.InputTypeInstanceRef
	}{
		"Should do nothing on nil": {
			givenIface:         nil,
			givenTypeInstances: nil,
		},
		"Should do nothing on zero values": {
			givenIface:         &gqlpublicapi.InterfaceRevision{},
			givenTypeInstances: []types.InputTypeInstanceRef{},
		},
	}
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			// given
			ctx := context.Background()

			validator := NewValidator(&fakeHubCli{})

			// when
			ifaceTypes, err := validator.LoadIfaceInputTypeInstanceRefs(ctx, tc.givenIface)
			// then
			require.NoError(t, err)
			require.Len(t, ifaceTypes, 0)

			// when
			result, err := validator.ValidateTypeInstances(ctx, ifaceTypes, tc.givenTypeInstances)
			// then
			require.NoError(t, err)
			assert.NoError(t, result.ErrorOrNil())
		})
	}
}

var implementationRevisionRaw = []byte(`
revision: 0.1.0
spec:
  additionalInput:
    parameters:
    - name: additional-parameters
      typeRef:
        path: cap.type.aws.auth.creds
        revision: 0.1.0
    - name: impl-specific-config
      typeRef:
        path: cap.type.aws.elasticsearch.install-input
        revision: 0.1.0
`)

func TestValidateImplementationParameters(t *testing.T) {
	// given
	impl := gqlpublicapi.ImplementationRevision{}
	require.NoError(t, yaml.Unmarshal(implementationRevisionRaw, &impl))

	tests := map[string]struct {
		givenHubTypeInstances []*gqlpublicapi.TypeRevision
		givenParameters       types.ParametersCollection
		expectedIssues        string
	}{
		"Happy path JSON": {
			givenHubTypeInstances: []*gqlpublicapi.TypeRevision{
				fixAWSCredsTypeRev(),
				fixAWSElasticsearchTypeRev(),
			},
			givenParameters: types.ParametersCollection{
				"additional-parameters": `{"key": "true"}`,
				"impl-specific-config":  `{"replicas": "3"}`,
			},
		},
		"Happy path YAML": {
			givenHubTypeInstances: []*gqlpublicapi.TypeRevision{
				fixAWSCredsTypeRev(),
				fixAWSElasticsearchTypeRev(),
			},
			givenParameters: types.ParametersCollection{
				"additional-parameters": `key: "true"`,
				"impl-specific-config":  `replicas: "3"`,
			},
		},
		"Not found `db-settings`": {
			givenHubTypeInstances: []*gqlpublicapi.TypeRevision{
				fixAWSCredsTypeRev(),
				fixAWSElasticsearchTypeRev(),
			},
			givenParameters: types.ParametersCollection{
				"db-settings": `{"key": true}`,
			},
			expectedIssues: heredoc.Doc(`
			    - Parameters "db-settings":
			        * Unknown parameter. Cannot validate it against JSONSchema.`),
		},
		"Invalid parameters": {
			givenHubTypeInstances: []*gqlpublicapi.TypeRevision{
				fixAWSCredsTypeRev(),
				fixAWSElasticsearchTypeRev(),
			},
			givenParameters: types.ParametersCollection{
				"additional-parameters": `{"key": true}`,
				"impl-specific-config":  `{"key": true}`,
			},
			expectedIssues: heredoc.Doc(`
			            	- Parameters "additional-parameters":
			            	    * key: Invalid type. Expected: string, given: boolean
			            	- Parameters "impl-specific-config":
			            	    * (root): replicas is required
			            	    * (root): Additional property key is not allowed`),
		},
	}
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			// given
			ctx := context.Background()
			fakeCli := &fakeHubCli{
				Types: tc.givenHubTypeInstances,
			}

			validator := NewValidator(fakeCli)

			// when
			implSchemas, err := validator.LoadImplInputParametersSchemas(ctx, impl)
			// then
			require.NoError(t, err)
			require.Len(t, implSchemas, 2)

			// when
			result, err := validator.ValidateParameters(ctx, implSchemas, tc.givenParameters)
			// then
			require.NoError(t, err)

			if tc.expectedIssues == "" {
				assert.NoError(t, result.ErrorOrNil())
			} else {
				assert.EqualError(t, result.ErrorOrNil(), tc.expectedIssues)
			}
		})
	}
}

func TestResolveTypeRefsToJSONSchemasFailures(t *testing.T) {
	// given
	tests := map[string]struct {
		givenTypeRefs                           validation.TypeRefCollection
		givenHubTypeInstances                   []*gqlpublicapi.TypeRevision
		givenListTypeRefRevisionsJSONSchemasErr error
		expectedErrorMsg                        string
	}{
		"Not existing TypeRef": {
			givenHubTypeInstances: nil,
			givenTypeRefs: validation.TypeRefCollection{
				"aws-creds": {
					TypeRef: types.TypeRef{
						Path:     "cap.type.aws.auth.creds",
						Revision: "0.1.0",
					},
				},
			},
			expectedErrorMsg: heredoc.Doc(`
		          1 error occurred:
		          	* TypeRef "cap.type.aws.auth.creds:0.1.0" was not found in Hub`),
		},
		"Not existing Revision": {
			givenHubTypeInstances: []*gqlpublicapi.TypeRevision{
				fixAWSCredsTypeRev(),
			},
			givenTypeRefs: validation.TypeRefCollection{
				"aws-creds": {
					TypeRef: types.TypeRef{
						Path:     "cap.type.aws.auth.creds",
						Revision: "1.1.1",
					},
				},
			},
			expectedErrorMsg: heredoc.Doc(`
		          1 error occurred:
		          	* TypeRef "cap.type.aws.auth.creds:1.1.1" was not found in Hub`),
		},
		"Unexpected JSONSchema type": {
			givenHubTypeInstances: []*gqlpublicapi.TypeRevision{
				func() *gqlpublicapi.TypeRevision {
					ti := fixAWSCredsTypeRev()
					ti.Spec.JSONSchema = 123 // change type to int, but should be string
					return ti
				}(),
			},
			givenTypeRefs: validation.TypeRefCollection{
				"aws-creds": {
					TypeRef: types.TypeRef{
						Path:     "cap.type.aws.auth.creds",
						Revision: "0.1.0",
					},
				},
			},
			expectedErrorMsg: heredoc.Doc(`
		          1 error occurred:
		          	* unexpected JSONSchema type for "cap.type.aws.auth.creds:0.1.0": expected string, got int`),
		},
		"Hub call error": {
			givenListTypeRefRevisionsJSONSchemasErr: errors.New("hub error for testing purposes"),
			givenTypeRefs: validation.TypeRefCollection{
				"aws-creds": {},
			},
			expectedErrorMsg: "while fetching JSONSchemas for input TypeRefs: hub error for testing purposes",
		},
	}
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			// given
			ctx := context.Background()
			fakeCli := &fakeHubCli{
				Types:                                tc.givenHubTypeInstances,
				ListTypeRefRevisionsJSONSchemasError: tc.givenListTypeRefRevisionsJSONSchemasErr,
			}

			validator := NewValidator(fakeCli)

			// when
			_, err := validator.resolveTypeRefsToJSONSchemas(ctx, tc.givenTypeRefs)

			// then
			assert.EqualError(t, err, tc.expectedErrorMsg)
		})
	}
}

type fakeHubCli struct {
	Types                                []*gqlpublicapi.TypeRevision
	IDsTypeRefs                          map[string]gqllocalapi.TypeInstanceTypeReference
	ListTypeRefRevisionsJSONSchemasError error
}

func (f *fakeHubCli) FindTypeInstancesTypeRef(_ context.Context, ids []string) (map[string]gqllocalapi.TypeInstanceTypeReference, error) {
	return f.IDsTypeRefs, nil
}

func (f *fakeHubCli) ListTypeRefRevisionsJSONSchemas(_ context.Context, filter gqlpublicapi.TypeFilter) ([]*gqlpublicapi.TypeRevision, error) {
	return f.Types, f.ListTypeRefRevisionsJSONSchemasError
}

func fixAWSCredsTypeRev() *gqlpublicapi.TypeRevision {
	return &gqlpublicapi.TypeRevision{
		Metadata: &gqlpublicapi.TypeMetadata{
			Path: "cap.type.aws.auth.creds",
		},
		Revision: "0.1.0",
		Spec: &gqlpublicapi.TypeSpec{
			JSONSchema: heredoc.Doc(`
                    {
                      "$schema": "http://json-schema.org/draft-07/schema",
                      "type": "object",
                      "required": [ "key" ],
                      "properties": {
                        "key": {
                          "type": "string"
                        }
                      }
                    }`),
		},
	}
}

func fixAWSElasticsearchTypeRev() *gqlpublicapi.TypeRevision {
	return &gqlpublicapi.TypeRevision{
		Metadata: &gqlpublicapi.TypeMetadata{
			Path: "cap.type.aws.elasticsearch.install-input",
		},
		Revision: "0.1.0",
		Spec: &gqlpublicapi.TypeSpec{
			JSONSchema: heredoc.Doc(`
                    {
                      "$schema": "http://json-schema.org/draft-07/schema",
                      "type": "object",
                      "title": "The schema for Elasticsearch input parameters.",
                      "required": ["replicas"],
                      "properties": {
                        "replicas": {
                          "type": "string",
                          "title": "Replica count for the Elasticsearch"
                        }
                      },
                      "additionalProperties": false
                    }`),
		},
	}
}
