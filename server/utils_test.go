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
	_, err := makeRequest(context.Background(), *http.DefaultClient, http.MethodGet, "", x)
	require.Error(t, err)
}
