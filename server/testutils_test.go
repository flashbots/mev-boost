package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_hexToBytes(t *testing.T) {
	assert.Equal(t, []byte{0x01, 0x02}, _hexToBytes("0x0102"))
	require.Panics(t, func() {
		_hexToBytes("foo")
	})
}

func Test_mockRelay(t *testing.T) {
	t.Run("bad payload", func(t *testing.T) {
		relay := newMockRelay()
		req, err := http.NewRequest("POST", pathRegisterValidator, bytes.NewReader([]byte("123")))
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		relay.getRouter().ServeHTTP(rr, req)
		require.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
