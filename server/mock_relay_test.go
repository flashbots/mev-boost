package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_mockRelay(t *testing.T) {
	t.Run("bad payload", func(t *testing.T) {
		relay := newMockRelay(t)
		req, err := http.NewRequest(http.MethodPost, pathRegisterValidator, bytes.NewReader([]byte("123")))
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		relay.getRouter().ServeHTTP(rr, req)
		require.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
