package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/machinebox/graphql"
	"github.com/sanity-io/litter"
	"github.com/stretchr/testify/require"
)

func TestPolicy(t *testing.T) {
	fakeServer := server(t)
	defer fakeServer.Close()

	cli := Policy{client: graphql.NewClient(fakeServer.URL)}

	// when
	policy, err := cli.GetPolicy(context.Background())

	// then
	require.NoError(t, err)
	litter.Dump(policy.TypeInstance)
}

func server(t *testing.T) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//b, err := ioutil.ReadAll(r.Body)
		//require.NoError(t, err)

		//assert.Equal(t, `{"query":"query {}","variables":null}`+"\n", string(b))
		data, err := os.ReadFile("./testdata/policy.json")
		require.NoError(t, err)

		_, _ = w.Write(data)
	}))
	return srv
}
