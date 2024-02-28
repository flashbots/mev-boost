package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	builderApi "github.com/attestantio/go-builder-client/api"
	builderApiDeneb "github.com/attestantio/go-builder-client/api/deneb"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/flashbots/mev-boost/config"
	"github.com/stretchr/testify/require"
)

func TestMakePostRequest(t *testing.T) {
	// Test errors
	var x chan bool
	code, err := SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, "", "test", nil, x, nil)
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
	ts := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		require.Equal(t, expectedUA, r.Header.Get("User-Agent"))
		done <- true
	}))
	code, err := SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, ts.URL, UserAgent(customUA), nil, nil, nil)
	ts.Close()
	require.NoError(t, err)
	require.Equal(t, 200, code)
	<-done

	// Test without custom UA
	expectedUA = fmt.Sprintf("mev-boost/%s", config.Version)
	ts = httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		require.Equal(t, expectedUA, r.Header.Get("User-Agent"))
		done <- true
	}))
	code, err = SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, ts.URL, "", nil, nil, nil)
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
	code, err := SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, ts.URL, "", nil, nil, &resp)
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

func TestGetPayloadResponseIsEmpty(t *testing.T) {
	testCases := []struct {
		name     string
		payload  *builderApi.VersionedSubmitBlindedBlockResponse
		expected bool
	}{
		{
			name: "Non-empty capella payload response",
			payload: &builderApi.VersionedSubmitBlindedBlockResponse{
				Version: spec.DataVersionCapella,
				Capella: &capella.ExecutionPayload{
					BlockHash: phase0.Hash32{0x1},
				},
			},
			expected: false,
		},
		{
			name: "Non-empty deneb payload response",
			payload: &builderApi.VersionedSubmitBlindedBlockResponse{
				Version: spec.DataVersionDeneb,
				Deneb: &builderApiDeneb.ExecutionPayloadAndBlobsBundle{
					ExecutionPayload: &deneb.ExecutionPayload{
						BlockHash: phase0.Hash32{0x1},
					},
					BlobsBundle: &builderApiDeneb.BlobsBundle{
						Blobs:       make([]deneb.Blob, 0),
						Commitments: make([]deneb.KZGCommitment, 0),
						Proofs:      make([]deneb.KZGProof, 0),
					},
				},
			},
			expected: false,
		},
		{
			name: "Empty capella payload response",
			payload: &builderApi.VersionedSubmitBlindedBlockResponse{
				Version: spec.DataVersionCapella,
			},
			expected: true,
		},
		{
			name: "Nil block hash for capella payload response",
			payload: &builderApi.VersionedSubmitBlindedBlockResponse{
				Version: spec.DataVersionCapella,
				Capella: &capella.ExecutionPayload{
					BlockHash: nilHash,
				},
			},
			expected: true,
		},
		{
			name: "Empty deneb payload response",
			payload: &builderApi.VersionedSubmitBlindedBlockResponse{
				Version: spec.DataVersionDeneb,
			},
			expected: true,
		},
		{
			name: "Empty deneb execution payload",
			payload: &builderApi.VersionedSubmitBlindedBlockResponse{
				Version: spec.DataVersionDeneb,
				Deneb: &builderApiDeneb.ExecutionPayloadAndBlobsBundle{
					BlobsBundle: &builderApiDeneb.BlobsBundle{
						Blobs:       make([]deneb.Blob, 0),
						Commitments: make([]deneb.KZGCommitment, 0),
						Proofs:      make([]deneb.KZGProof, 0),
					},
				},
			},
			expected: true,
		},
		{
			name: "Empty deneb blobs bundle",
			payload: &builderApi.VersionedSubmitBlindedBlockResponse{
				Deneb: &builderApiDeneb.ExecutionPayloadAndBlobsBundle{
					ExecutionPayload: &deneb.ExecutionPayload{
						BlockHash: phase0.Hash32{0x1},
					},
				},
			},
			expected: true,
		},
		{
			name: "Nil block hash for deneb payload response",
			payload: &builderApi.VersionedSubmitBlindedBlockResponse{
				Deneb: &builderApiDeneb.ExecutionPayloadAndBlobsBundle{
					ExecutionPayload: &deneb.ExecutionPayload{
						BlockHash: nilHash,
					},
				},
			},
			expected: true,
		},
		{
			name: "Unsupported payload version",
			payload: &builderApi.VersionedSubmitBlindedBlockResponse{
				Version: spec.DataVersionBellatrix,
			},
			expected: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, getPayloadResponseIsEmpty(tt.payload))
		})
	}
}
