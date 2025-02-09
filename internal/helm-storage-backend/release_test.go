package helmstoragebackend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	"helm.sh/helm/v3/pkg/time"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"capact.io/capact/internal/logger"
	"capact.io/capact/internal/ptr"
	pb "capact.io/capact/pkg/hub/api/grpc/storage_backend"
)

func TestRelease_GetValue_Success(t *testing.T) {
	tests := []struct {
		name string

		givenDriver          *string
		givenTypeInstanceID  string
		givenResourceVersion uint32
		expectedDriver       string
	}{
		{
			name:                "should use default driver and return the latest release",
			givenTypeInstanceID: "123",
			givenDriver:         nil,
			expectedDriver:      "secrets",
		},
		{
			name:                "should use configmap driver and return the latest release",
			givenTypeInstanceID: "123",
			givenDriver:         ptr.String("configmaps"),
			expectedDriver:      "configmaps",
		},
		{
			name:                 "should ignore resourceVersion and return the latest release",
			givenTypeInstanceID:  "123",
			givenResourceVersion: 42, // should be ignored
			expectedDriver:       "secrets",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			// given
			const (
				releaseName      = "test-get-release"
				releaseNamespace = "test-get-namespace"
				chartLocation    = "http://example.com/charts"
			)
			expHelmRelease := fixHelmRelease(releaseName, releaseNamespace)
			expFlags := &genericclioptions.ConfigFlags{ClusterName: ptr.String("testing")}
			mockConfigurationProducer := mockConfigurationProducer(t, expHelmRelease, expFlags, test.expectedDriver)

			givenReq := &pb.GetValueRequest{
				TypeInstanceId:  test.givenTypeInstanceID,
				ResourceVersion: test.givenResourceVersion,
				Context: mustMarshal(t, ReleaseContext{
					Name:          releaseName,
					Namespace:     releaseNamespace,
					ChartLocation: chartLocation,
					Driver:        test.givenDriver,
				}),
			}

			expResponse := &pb.GetValueResponse{
				Value: mustMarshal(t, ReleaseDetails{
					Name:      expHelmRelease.Name,
					Namespace: expHelmRelease.Namespace,
					Chart: ChartDetails{
						Name:    expHelmRelease.Chart.Metadata.Name,
						Version: expHelmRelease.Chart.Metadata.Version,
						Repo:    chartLocation,
					},
				}),
			}

			svc, err := NewReleaseHandler(logger.Noop(), expFlags)
			svc.actionConfigurationProducer = mockConfigurationProducer
			require.NoError(t, err)

			// when
			outVal, gotErr := svc.GetValue(context.Background(), givenReq)

			// then
			assert.NoError(t, gotErr)
			assert.Equal(t, outVal, expResponse)
		})
	}
}

func TestRelease_GetValue_Failures(t *testing.T) {
	// globally given
	const (
		releaseName      = "test-release"
		releaseNamespace = "test-namespace"
	)
	tests := []struct {
		name string

		request       *pb.GetValueRequest
		internalError error

		expErrMsg string
	}{
		{
			name: "should return not found error if release name is wrong",
			request: &pb.GetValueRequest{
				TypeInstanceId: "123",
				Context: mustMarshal(t, ReleaseContext{
					Name:          "other-release",
					Namespace:     releaseNamespace,
					ChartLocation: "http://example.com/charts",
				}),
			},
			expErrMsg: "rpc error: code = NotFound desc = Helm release 'test-namespace/other-release' for TypeInstance '123' was not found",
		},
		{
			name: "should return not found error if release namespace is wrong",
			request: &pb.GetValueRequest{
				TypeInstanceId: "123",
				Context: mustMarshal(t, ReleaseContext{
					Name:          releaseName,
					Namespace:     "other-ns",
					ChartLocation: "http://example.com/charts",
				}),
			},
			expErrMsg: "rpc error: code = NotFound desc = Helm release 'other-ns/test-release' for TypeInstance '123' was not found",
		},
		{
			name: "should return internal error",
			request: &pb.GetValueRequest{
				TypeInstanceId: "123",
				Context: mustMarshal(t, ReleaseContext{
					Name:          releaseName,
					Namespace:     "other-ns",
					ChartLocation: "http://example.com/charts",
				}),
			},
			internalError: errors.New("internal error"),
			expErrMsg:     "rpc error: code = Internal desc = while creating Helm get release client: internal error",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			// given
			expHelmRelease := fixHelmRelease(releaseName, releaseNamespace)
			expFlags := &genericclioptions.ConfigFlags{ClusterName: ptr.String("testing")}

			mockConfigurationProducer := func(inputFlags *genericclioptions.ConfigFlags, inputDriver, inputNs string) (*action.Configuration, error) {
				if test.internalError != nil {
					return nil, test.internalError
				}
				producer := mockConfigurationProducer(t, expHelmRelease, expFlags, "secrets")
				return producer(inputFlags, inputDriver, inputNs)
			}

			svc, err := NewReleaseHandler(logger.Noop(), expFlags)
			svc.actionConfigurationProducer = mockConfigurationProducer
			require.NoError(t, err)

			// when
			outVal, gotErr := svc.GetValue(context.Background(), test.request)

			// then
			assert.EqualError(t, gotErr, test.expErrMsg)
			assert.Nil(t, outVal)
		})
	}
}

func TestRelease_OnCreate_Success(t *testing.T) {
	// given
	const (
		releaseName      = "test-create-release"
		releaseNamespace = "test-create-namespace"
		releaseDriver    = "configmap"
		chartLocation    = "http://example.com/charts"
	)
	expHelmRelease := fixHelmRelease(releaseName, releaseNamespace)
	expFlags := &genericclioptions.ConfigFlags{ClusterName: ptr.String("testing")}
	mockConfigurationProducer := mockConfigurationProducer(t, expHelmRelease, expFlags, releaseDriver)

	givenReq := &pb.OnCreateRequest{
		TypeInstanceId: "42",
		Context: mustMarshal(t, ReleaseContext{
			Name:          releaseName,
			Namespace:     releaseNamespace,
			ChartLocation: chartLocation,
			Driver:        ptr.String(releaseDriver),
		}),
	}

	svc, err := NewReleaseHandler(logger.Noop(), expFlags)
	svc.actionConfigurationProducer = mockConfigurationProducer
	require.NoError(t, err)

	// when
	gotOut, gotErr := svc.OnCreate(context.Background(), givenReq)

	// then
	assert.NoError(t, gotErr)
	assert.Empty(t, gotOut)
}

func TestRelease_OnCreate_Failures(t *testing.T) {
	// globally given
	const (
		releaseName      = "test-release"
		releaseNamespace = "test-namespace"
	)
	tests := []struct {
		name string

		request       *pb.OnCreateRequest
		internalError error

		expErrMsg string
	}{
		{
			name: "should return not found error if release name is wrong",
			request: &pb.OnCreateRequest{
				TypeInstanceId: "123",
				Context: mustMarshal(t, ReleaseContext{
					Name:          "other-release",
					Namespace:     releaseNamespace,
					ChartLocation: "http://example.com/charts",
				}),
			},
			expErrMsg: "rpc error: code = NotFound desc = Helm release 'test-namespace/other-release' for TypeInstance '123' was not found",
		},
		{
			name: "should return not found error if release namespace is wrong",
			request: &pb.OnCreateRequest{
				TypeInstanceId: "123",
				Context: mustMarshal(t, ReleaseContext{
					Name:          releaseName,
					Namespace:     "other-ns",
					ChartLocation: "http://example.com/charts",
				}),
			},
			expErrMsg: "rpc error: code = NotFound desc = Helm release 'other-ns/test-release' for TypeInstance '123' was not found",
		},
		{
			name: "should return internal error",
			request: &pb.OnCreateRequest{
				TypeInstanceId: "123",
				Context: mustMarshal(t, ReleaseContext{
					Name:          releaseName,
					Namespace:     releaseNamespace,
					ChartLocation: "http://example.com/charts",
				}),
			},
			internalError: errors.New("internal error"),
			expErrMsg:     "rpc error: code = Internal desc = while creating Helm get release client: internal error",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			// given
			expHelmRelease := fixHelmRelease(releaseName, releaseNamespace)
			expFlags := &genericclioptions.ConfigFlags{ClusterName: ptr.String("testing")}

			mockConfigurationProducer := func(inputFlags *genericclioptions.ConfigFlags, inputDriver, inputNs string) (*action.Configuration, error) {
				if test.internalError != nil {
					return nil, test.internalError
				}
				producer := mockConfigurationProducer(t, expHelmRelease, expFlags, "secrets")
				return producer(inputFlags, inputDriver, inputNs)
			}

			svc, err := NewReleaseHandler(logger.Noop(), expFlags)
			svc.actionConfigurationProducer = mockConfigurationProducer
			require.NoError(t, err)

			// when
			outVal, gotErr := svc.OnCreate(context.Background(), test.request)

			// then
			assert.EqualError(t, gotErr, test.expErrMsg)
			assert.Nil(t, outVal)
		})
	}
}

func TestRelease_OnUpdate_Success(t *testing.T) {
	// given
	const (
		releaseName      = "test-update-release"
		releaseNamespace = "test-update-namespace"
		releaseDriver    = "configmap"
		chartLocation    = "http://example.com/charts"
	)
	expHelmRelease := fixHelmRelease(releaseName, releaseNamespace)
	expFlags := &genericclioptions.ConfigFlags{ClusterName: ptr.String("testing")}
	mockConfigurationProducer := mockConfigurationProducer(t, expHelmRelease, expFlags, releaseDriver)

	givenReq := &pb.OnUpdateRequest{
		TypeInstanceId: "42",
		Context: mustMarshal(t, ReleaseContext{
			Name:          releaseName,
			Namespace:     releaseNamespace,
			ChartLocation: chartLocation,
			Driver:        ptr.String(releaseDriver),
		}),
	}

	svc, err := NewReleaseHandler(logger.Noop(), expFlags)
	svc.actionConfigurationProducer = mockConfigurationProducer
	require.NoError(t, err)

	// when
	gotOut, gotErr := svc.OnUpdate(context.Background(), givenReq)

	// then
	assert.NoError(t, gotErr)
	assert.Empty(t, gotOut)
}

func TestRelease_OnUpdate_Failures(t *testing.T) {
	// globally given
	const (
		releaseName      = "test-release"
		releaseNamespace = "test-namespace"
	)
	tests := []struct {
		name string

		request       *pb.OnUpdateRequest
		internalError error

		expErrMsg string
	}{
		{
			name: "should return not found error if release name is wrong",
			request: &pb.OnUpdateRequest{
				TypeInstanceId: "123",
				Context: mustMarshal(t, ReleaseContext{
					Name:          "other-release",
					Namespace:     releaseNamespace,
					ChartLocation: "http://example.com/charts",
				}),
			},
			expErrMsg: "rpc error: code = NotFound desc = Helm release 'test-namespace/other-release' for TypeInstance '123' was not found",
		},
		{
			name: "should return not found error if release namespace is wrong",
			request: &pb.OnUpdateRequest{
				TypeInstanceId: "123",
				Context: mustMarshal(t, ReleaseContext{
					Name:          releaseName,
					Namespace:     "other-ns",
					ChartLocation: "http://example.com/charts",
				}),
			},
			expErrMsg: "rpc error: code = NotFound desc = Helm release 'other-ns/test-release' for TypeInstance '123' was not found",
		},
		{
			name: "should return internal error",
			request: &pb.OnUpdateRequest{
				TypeInstanceId: "123",
				Context: mustMarshal(t, ReleaseContext{
					Name:          releaseName,
					Namespace:     releaseNamespace,
					ChartLocation: "http://example.com/charts",
				}),
			},
			internalError: errors.New("internal error"),
			expErrMsg:     "rpc error: code = Internal desc = while creating Helm get release client: internal error",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			// given
			expHelmRelease := fixHelmRelease(releaseName, releaseNamespace)
			expFlags := &genericclioptions.ConfigFlags{ClusterName: ptr.String("testing")}

			mockConfigurationProducer := func(inputFlags *genericclioptions.ConfigFlags, inputDriver, inputNs string) (*action.Configuration, error) {
				if test.internalError != nil {
					return nil, test.internalError
				}
				producer := mockConfigurationProducer(t, expHelmRelease, expFlags, "secrets")
				return producer(inputFlags, inputDriver, inputNs)
			}

			svc, err := NewReleaseHandler(logger.Noop(), expFlags)
			svc.actionConfigurationProducer = mockConfigurationProducer
			require.NoError(t, err)

			// when
			outVal, gotErr := svc.OnUpdate(context.Background(), test.request)

			// then
			assert.EqualError(t, gotErr, test.expErrMsg)
			assert.Nil(t, outVal)
		})
	}
}

func TestRelease_NOP_Methods(t *testing.T) {
	// globally given
	tests := []struct {
		name    string
		handler func(ctx context.Context, svc *ReleaseHandler) (interface{}, error)
	}{
		{
			name: "no operation for OnDelete",
			handler: func(ctx context.Context, svc *ReleaseHandler) (interface{}, error) {
				return svc.OnDelete(ctx, nil)
			},
		},
		{
			name: "no operation for GetLockedBy",
			handler: func(ctx context.Context, svc *ReleaseHandler) (interface{}, error) {
				return svc.GetLockedBy(ctx, nil)
			},
		},
		{
			name: "no operation for OnLock",
			handler: func(ctx context.Context, svc *ReleaseHandler) (interface{}, error) {
				return svc.OnLock(ctx, nil)
			},
		},
		{
			name: "no operation for OnUnlock",
			handler: func(ctx context.Context, svc *ReleaseHandler) (interface{}, error) {
				return svc.OnUnlock(ctx, nil)
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			// given
			producerCalled := false
			mockConfigurationProducer := func(_ *genericclioptions.ConfigFlags, _, _ string) (*action.Configuration, error) {
				producerCalled = true
				return nil, nil
			}
			svc, err := NewReleaseHandler(logger.Noop(), nil)
			svc.actionConfigurationProducer = mockConfigurationProducer
			require.NoError(t, err)

			// when
			outVal, gotErr := test.handler(context.Background(), svc)

			// then
			assert.NoError(t, gotErr)
			assert.False(t, producerCalled)
			assert.Empty(t, outVal)
		})
	}
}

func mockConfigurationProducer(t *testing.T, expHelmRelease *release.Release, expFlags *genericclioptions.ConfigFlags, expDriver string) actionConfigurationProducerFn {
	t.Helper()
	inMemoryDriver := driver.NewMemory()
	err := inMemoryDriver.Create("1", expHelmRelease)
	require.NoError(t, err)

	return func(inputFlags *genericclioptions.ConfigFlags, inputDriver, inputNs string) (*action.Configuration, error) {
		assert.Equal(t, expFlags, inputFlags)
		assert.Equal(t, expDriver, inputDriver)

		inMemoryDriver.SetNamespace(inputNs)
		return &action.Configuration{
			Releases:   storage.Init(inMemoryDriver),
			KubeClient: &kubefake.FailingKubeClient{PrintingKubeClient: kubefake.PrintingKubeClient{Out: ioutil.Discard}},
		}, nil
	}
}

func mustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()
	out, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func fixHelmRelease(name, ns string) *release.Release {
	now := time.Now()
	return &release.Release{
		Name:      name,
		Namespace: ns,
		Info: &release.Info{
			FirstDeployed: now,
			LastDeployed:  now,
			Description:   "Named Release Stub",
		},
		Chart: &chart.Chart{
			Metadata: &chart.Metadata{
				Name:    fmt.Sprintf("%s-chart", name),
				Version: "0.1.0",
			},
		},
	}
}
