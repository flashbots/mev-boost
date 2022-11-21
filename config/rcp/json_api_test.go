package rcp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flashbots/mev-boost/config/rcp"
	"github.com/flashbots/mev-boost/config/relay"
	"github.com/flashbots/mev-boost/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONAPIRelayConfigProvider(t *testing.T) {
	t.Parallel()

	t.Run("it returns configuration for a given validator by public key", func(t *testing.T) {
		t.Parallel()

		// arrange
		srv := httptest.NewServer(successfulHandler(testdata.CorrectRelayConfig))
		defer srv.Close()

		want := successfulResponse(t)
		sut := rcp.NewJSONAPI(http.DefaultClient, srv.URL)

		// act
		got, err := sut.FetchConfig()

		// assert
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("it returns an error if provider relayURL is malformed", func(t *testing.T) {
		t.Parallel()

		// arrange
		sut := rcp.NewJSONAPI(http.DefaultClient, "http://a b.com/")

		// act
		_, err := sut.FetchConfig()

		// assert
		var rcpError rcp.Error
		assert.ErrorAs(t, err, &rcpError)
		assert.ErrorIs(t, err, rcp.ErrMalformedProviderURL)
	})

	t.Run("it returns an error if it cannot fetch relays config", func(t *testing.T) {
		t.Parallel()

		// arrange
		sut := rcp.NewJSONAPI(http.DefaultClient, "http://invalid-url")

		// act
		_, err := sut.FetchConfig()

		// assert
		var rcpError rcp.Error
		assert.ErrorAs(t, err, &rcpError)
		assert.ErrorIs(t, err, rcp.ErrHTTPRequestFailed)
	})

	t.Run("it returns an error if malformed http response is returned with status code 200", func(t *testing.T) {
		t.Parallel()

		// arrange
		srv := httptest.NewServer(malformedHandler(http.StatusOK))
		defer srv.Close()

		sut := rcp.NewJSONAPI(http.DefaultClient, srv.URL)

		// act
		_, err := sut.FetchConfig()

		// assert
		var rcpError rcp.Error
		assert.ErrorAs(t, err, &rcpError)
		assert.ErrorIs(t, err, rcp.ErrMalformedResponseBody)
	})

	t.Run("it returns an error if malformed api error response is returned", func(t *testing.T) {
		t.Parallel()

		// arrange
		srv := httptest.NewServer(malformedHandler(http.StatusInternalServerError))
		defer srv.Close()

		sut := rcp.NewJSONAPI(http.DefaultClient, srv.URL)

		// act
		_, err := sut.FetchConfig()

		// assert
		var rcpError rcp.Error
		assert.ErrorAs(t, err, &rcpError)
		assert.ErrorIs(t, err, rcp.ErrMalformedResponseBody)
	})

	t.Run("it handles API errors", func(t *testing.T) {
		t.Parallel()

		// arrange
		srv := httptest.NewServer(errorHandler())
		defer srv.Close()

		sut := rcp.NewJSONAPI(http.DefaultClient, srv.URL)

		// act
		_, err := sut.FetchConfig()

		// assert
		var apiErr *rcp.APIError
		assert.ErrorAs(t, err, &apiErr)
		assert.ErrorIs(t, err, rcp.ErrCannotFetchRelays)
	})
}

func successfulResponse(t *testing.T) *relay.Config {
	t.Helper()

	var want *relay.Config
	require.NoError(t, json.Unmarshal(testdata.CorrectRelayConfig, &want))

	return want
}

func successfulHandler(body []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

func errorHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":500,"message":"Internal Server Error"}`))
	}
}

func malformedHandler(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"invalid json",`))
	}
}
