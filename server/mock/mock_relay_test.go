package mock

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flashbots/mev-boost/server/params"
	"github.com/stretchr/testify/require"
)

func Test_mockRelay(t *testing.T) {
	t.Run("bad payload", func(t *testing.T) {
		relay := NewRelay(t)
		req, err := http.NewRequest(http.MethodPost, params.PathRegisterValidator, bytes.NewReader([]byte("123")))
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		relay.getRouter().ServeHTTP(rr, req)
		require.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
