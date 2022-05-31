package server

import (
	"bytes"
	"github.com/flashbots/go-boost-utils/bls"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_mockRelay(t *testing.T) {
	t.Run("bad payload", func(t *testing.T) {
		privateKey, _, err := bls.GenerateNewKeypair()
		require.NoError(t, err)

		relay := NewMockRelay(t, privateKey)
		req, err := http.NewRequest("POST", pathRegisterValidator, bytes.NewReader([]byte("123")))
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		relay.getRouter().ServeHTTP(rr, req)
		require.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
