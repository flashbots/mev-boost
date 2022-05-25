package server

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMakePostRequest(t *testing.T) {
	// Test errors
	var x chan bool
	err := SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, "", x, nil)
	require.Error(t, err)
}
