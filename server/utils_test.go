package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/capella"
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
	code, err := SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, ts.URL, UserAgent(customUA), nil, nil)
	ts.Close()
	require.NoError(t, err)
	require.Equal(t, 200, code)
	<-done

	// Test without custom UA
	expectedUA = fmt.Sprintf("mev-boost/%s", config.Version)
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, expectedUA, r.Header.Get("User-Agent"))
		done <- true
	}))
	code, err = SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, ts.URL, "", nil, nil)
	ts.Close()
	require.NoError(t, err)
	require.Equal(t, 200, code)
	<-done
}

func TestSendHTTPRequestGzip(t *testing.T) {
	// Test with gzip response
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write([]byte(`{ "msg": "test-message" }`))
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "gzip", r.Header.Get("Accept-Encoding"))
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = w.Write(buf.Bytes())
	}))
	resp := struct{ Msg string }{}
	code, err := SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, ts.URL, "", nil, &resp)
	ts.Close()
	require.NoError(t, err)
	require.Equal(t, 200, code)
	require.Equal(t, "test-message", resp.Msg)
}

func TestWeiBigIntToEthBigFloat(t *testing.T) {
	// test with valid input
	i := big.NewInt(1)
	f := weiBigIntToEthBigFloat(i)
	require.Equal(t, "0.000000000000000001", f.Text('f', 18))

	// test with nil, which results on invalid big.Int input
	f = weiBigIntToEthBigFloat(nil)
	require.Equal(t, "0.000000000000000000", f.Text('f', 18))
}

func TestCapellaComputeBlockHash(t *testing.T) {
	jsonFile, err := os.Open("../testdata/zhejiang-execution-payload-capella.json")
	require.NoError(t, err)
	defer jsonFile.Close()

	payload := new(capella.ExecutionPayload)
	require.NoError(t, DecodeJSON(jsonFile, payload))

	hash, err := ComputeBlockHash(payload)
	require.NoError(t, err)
	require.Equal(t, "0x08751ea2076d3ecc606231495a90ba91a66a9b8fb1a2b76c333f1957a1c667c3", hash.String())
}
