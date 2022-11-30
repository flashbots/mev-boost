package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/bellatrix"
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

func TestCalculateBlockHash(t *testing.T) {
	payloadJSON := `{
		"parent_hash": "0x89236ba32cb76b3f17cbba7620d956d561a08a42a22145bb5705099ed94eaddf",
		"fee_recipient": "0x0000000000000000000000000000000000000000",
		"state_root": "0x44f451f33692bc78735f7836ad9c25761ba15609155e7bfcb38ded400d95d500",
		"receipts_root": "0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2",
		"logs_bloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		"prev_randao": "0x0000000000000000000000000000000000000000000000000000000000000000",
		"block_number": "11",
		"gas_limit": "4707788",
		"gas_used": "21000",
		"timestamp": "9155",
		"extra_data": "0x",
		"base_fee_per_gas": "233138867",
		"block_hash": "0x6662fb418aa7b5c5c80e2e8bc87be48db82e799c4704368d34ddeb3b12549655",
		"transactions": [
		  "0xf8670a843b9aca008252089400000000000000000000000000000000000000008203e880820a95a0ee3d06deddd2465aaa24bac5d329d3c40571c156c18d35c09a7c1daef2e95755a063e676889bbbdd27ab4e798b570f14ed8db32e4be22db15ab9f869c353b21f19"
		]
	}`
	payload := new(bellatrix.ExecutionPayload)
	require.NoError(t, DecodeJSON(strings.NewReader(payloadJSON), payload))

	hash, err := ComputeBlockHash(payload)
	require.NoError(t, err)
	require.Equal(t, "0x6662fb418aa7b5c5c80e2e8bc87be48db82e799c4704368d34ddeb3b12549655", hash.String())
}
