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
		assert.ErrorIs(t, err, rcp.ErrCannotFetchConfig)
	})

	t.Run("it uses a default http.Client if no configured client is passed", func(t *testing.T) {
		t.Parallel()

		// arrange
		srv := httptest.NewServer(successfulHandler(testdata.CorrectRelayConfig))
		defer srv.Close()

		want := successfulResponse(t)
		sut := rcp.NewJSONAPI(nil, srv.URL)

		// act
		got, err := sut.FetchConfig()

		// assert
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}

func TestError(t *testing.T) {
	t.Parallel()

	t.Run("test rcp error", func(t *testing.T) {
		t.Parallel()

		// arrange
		want := "cannot fetch relay config: malformed response body"

		// act
		err := rcp.WrapErr(rcp.ErrMalformedResponseBody)

		// assert
		var got rcp.Error
		require.ErrorAs(t, err, &got)
		assert.ErrorIs(t, err, rcp.ErrMalformedResponseBody)
		assert.Equal(t, err.Error(), want)
	})

	t.Run("test json api error", func(t *testing.T) {
		t.Parallel()

		// arrange
		want := "api error: 500: Internal Server Error: cannot fetch relay config"

		// act
		err := rcp.APIError{
			Code:    http.StatusInternalServerError,
			Message: http.StatusText(http.StatusInternalServerError),
		}

		// assert
		assert.ErrorIs(t, err, rcp.ErrCannotFetchConfig)
		assert.Equal(t, http.StatusInternalServerError, err.Code)
		assert.Equal(t, http.StatusText(http.StatusInternalServerError), err.Message)
		assert.Equal(t, want, err.Error())
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
