package rcp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flashbots/mev-boost/config/rcp"
	"github.com/flashbots/mev-boost/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONAPIRelayConfigProvider(t *testing.T) {
	t.Parallel()

	t.Run("it returns configuration for a given validator by public key", func(t *testing.T) {
		t.Parallel()

		// arrange
		srv := httptest.NewServer(successfulHandler(testdata.ValidProposerConfigBytes))
		defer srv.Close()

		want := testdata.ValidProposerConfig(t)
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
		assertRCPError(t, rcp.ErrMalformedProviderURL, err)
	})

	t.Run("it returns an error if it cannot fetch relays config", func(t *testing.T) {
		t.Parallel()

		// arrange
		sut := rcp.NewJSONAPI(http.DefaultClient, "http://invalid-url")

		// act
		_, err := sut.FetchConfig()

		// assert
		assertRCPError(t, rcp.ErrHTTPRequestFailed, err)
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
		assertRCPError(t, rcp.ErrMalformedResponseBody, err)
	})

	t.Run("it returns an error if an api error response is not a valid json", func(t *testing.T) {
		t.Parallel()

		// arrange
		srv := httptest.NewServer(malformedHandler(http.StatusInternalServerError))
		defer srv.Close()

		sut := rcp.NewJSONAPI(http.DefaultClient, srv.URL)

		// act
		_, err := sut.FetchConfig()

		// assert
		assertRCPError(t, rcp.ErrMalformedResponseBody, err)
	})

	t.Run("it handles api errors", func(t *testing.T) {
		t.Parallel()

		// arrange
		srv := httptest.NewServer(errorHandler())
		defer srv.Close()

		sut := rcp.NewJSONAPI(http.DefaultClient, srv.URL)

		// act
		_, err := sut.FetchConfig()

		// assert
		assertAPIError(t, rcp.ErrCannotFetchConfig, err)
	})

	t.Run("it uses a default http.Client if no configured client is passed", func(t *testing.T) {
		t.Parallel()

		// arrange
		srv := httptest.NewServer(successfulHandler(testdata.ValidProposerConfigBytes))
		defer srv.Close()

		want := testdata.ValidProposerConfig(t)
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
		assertRCPError(t, rcp.ErrMalformedResponseBody, err)
		assert.Equal(t, want, err.Error())
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
		assertAPIError(t, rcp.ErrCannotFetchConfig, err)
		assert.Equal(t, want, err.Error())
	})
}

func assertRCPError(t *testing.T, want, got error) {
	t.Helper()

	var rcpError rcp.Error
	assert.ErrorAs(t, got, &rcpError)
	assert.ErrorIs(t, got, want)
}

func assertAPIError(t *testing.T, want, got error) {
	t.Helper()

	var apiErr rcp.APIError
	assert.ErrorAs(t, got, &apiErr)
	assert.ErrorIs(t, got, want)

	assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
	assert.Equal(t, http.StatusText(http.StatusInternalServerError), apiErr.Message)
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
