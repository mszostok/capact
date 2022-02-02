package types_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	"capact.io/capact/pkg/sdk/apis/0.0.1/types"
)

type marshaler interface {
	Marshal() ([]byte, error)
}

func TestUnmarshalAndMarshalActionProduceSameResults(t *testing.T) {
	mustChDirToRoot(t)

	tests := map[string]struct {
		examplePath     string
		unmarshalMethod func(data []byte) (marshaler, error)
	}{
		"Implementation": {
			examplePath: "implementation.yaml",
			unmarshalMethod: func(data []byte) (marshaler, error) {
				obj, err := types.UnmarshalImplementation(data)
				if err != nil {
					return nil, err
				}
				return &obj, nil
			},
		},
		"Interface": {
			examplePath: "interface.yaml",
			unmarshalMethod: func(data []byte) (marshaler, error) {
				obj, err := types.UnmarshalInterface(data)
				if err != nil {
					return nil, err
				}
				return &obj, nil
			},
		},
		"RepoMetadata": {
			examplePath: "repo-metadata.yaml",
			unmarshalMethod: func(data []byte) (marshaler, error) {
				obj, err := types.UnmarshalRepoMetadata(data)
				if err != nil {
					return nil, err
				}
				return &obj, nil
			},
		},
		"Attribute": {
			examplePath: "attribute.yaml",
			unmarshalMethod: func(data []byte) (marshaler, error) {
				obj, err := types.UnmarshalAttribute(data)
				if err != nil {
					return nil, err
				}
				return &obj, nil
			},
		},
		"Type": {
			examplePath: "type.yaml",
			unmarshalMethod: func(data []byte) (marshaler, error) {
				obj, err := types.UnmarshalType(data)
				if err != nil {
					return nil, err
				}
				return &obj, nil
			},
		},
		"Vendor": {
			examplePath: "vendor.yaml",
			unmarshalMethod: func(data []byte) (marshaler, error) {
				obj, err := types.UnmarshalVendor(data)
				if err != nil {
					return nil, err
				}
				return &obj, nil
			},
		},
	}
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			// given
			buf, err := ioutil.ReadFile(path.Join("./ocf-spec/0.0.1/examples/", tc.examplePath))
			require.NoError(t, err, "while reading example file")

			buf, err = yaml.YAMLToJSON(buf)
			require.NoError(t, err, "while converting YAML to JSON")

			// when
			gotUnmarshal, err := tc.unmarshalMethod(buf)
			require.NoError(t, err, "while unmarshaling example file")

			gotMarshal, err := gotUnmarshal.Marshal()
			require.NoError(t, err, "while marshaling example file")

			// then
			// TODO: we can have a missing field with that assertion, should be fixed later.
			assert.JSONEq(t, string(buf), string(gotMarshal))
		})
	}
}

func mustChDirToRoot(t *testing.T) {
	t.Helper()

	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "../../../../../")
	err := os.Chdir(dir)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestTypeInstanceBackendCollection_Get(t *testing.T) {
	data := types.TypeInstanceBackendCollection{}

	data.SetByTypeRef(types.TypeRef{
		Path:     "cap.type.capactio.examples.message",
		Revision: "0.1.0",
	}, types.TypeInstanceBackend{
		ID: "1",
	})

	data.SetByTypeRef(types.TypeRef{
		Path: "cap.type.capactio.examples.*",
	}, types.TypeInstanceBackend{
		ID: "2",
	})

	data.SetByTypeRef(types.TypeRef{
		Path: "cap.*",
	}, types.TypeInstanceBackend{
		ID: "3",
	})

	fmt.Println(data.GetByTypeRef(types.TypeRef{
		Path:     "cap.capactio.examples.asdf",
		Revision: "0.1.0",
	}))
}
