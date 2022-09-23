package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flashbots/mev-boost/config"
	"github.com/stretchr/testify/require"
)

func TestMakePostRequest(t *testing.T) {
	// Test errors
	var x chan bool
	code, err := SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, "", "test", x, nil)
	require.Error(t, err)
	require.Equal(t, 0, code)
}

func TestDecodeJSON(t *testing.T) {
	// test disallows unknown fields
	var x struct {
		A int `json:"a"`
		B int `json:"b"`
	}
	payload := bytes.NewReader([]byte(`{"a":1,"b":2,"c":3}`))
	err := DecodeJSON(payload, &x)
	require.Error(t, err)
	require.Equal(t, "json: unknown field \"c\"", err.Error())
}

func TestSendHTTPRequestUserAgent(t *testing.T) {
	done := make(chan bool, 1)

	// Test with custom UA
	customUA := "test-user-agent"
	expectedUA := fmt.Sprintf("mev-boost/%s %s", config.Version, customUA)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, expectedUA, r.Header.Get("User-Agent"))
		done <- true
	}))
	defer ts.Close()
	code, err := SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, ts.URL, UserAgent(customUA), nil, nil)
	require.NoError(t, err)
	require.Equal(t, 200, code)
	<-done

	// Test without custom UA
	expectedUA = fmt.Sprintf("mev-boost/%s", config.Version)
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, expectedUA, r.Header.Get("User-Agent"))
		done <- true
	}))
	defer ts.Close()
	code, err = SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, ts.URL, "", nil, nil)
	require.NoError(t, err)
	require.Equal(t, 200, code)
	<-done
}

func Test_filterPath(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "metricOpts",
			args: args{
				path: pathMetrics,
			},
			want: pathMetrics,
		},
		{
			name: "status",
			args: args{
				path: pathStatus,
			},
			want: pathStatus,
		},
		{
			name: "getPayload",
			args: args{
				path: pathGetPayload,
			},
			want: pathGetPayload,
		},
		{
			name: "getHeader",
			args: args{
				path: pathGetHeader,
			},
			want: "/eth/v1/builder/header",
		},
		{
			name: "registerValidator",
			args: args{
				path: pathRegisterValidator,
			},
			want: pathRegisterValidator,
		},
		{
			name: "registerValidator full path",
			args: args{
				path: "https://builder-relay-goerli.flashbots.net/eth/v1/builder/header",
			},
			want: "https://builder-relay-goerli.flashbots.net/eth/v1/builder/header",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterPath(tt.args.path); got != tt.want {
				t.Errorf("filterPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
