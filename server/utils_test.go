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
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	t.Run("Non-empty capella payload response", func(t *testing.T) {
		payload := &builderApi.VersionedSubmitBlindedBlockResponse{
			Version: spec.DataVersionCapella,
			Capella: &capella.ExecutionPayload{
				BlockHash: phase0.Hash32{0x1},
			},
		}
		require.False(t, getPayloadResponseIsEmpty(payload))
	})

	t.Run("Non-empty deneb payload response", func(t *testing.T) {
		payload := &builderApi.VersionedSubmitBlindedBlockResponse{
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
		}
		require.False(t, getPayloadResponseIsEmpty(payload))
	})

	t.Run("Empty capella payload response", func(t *testing.T) {
		payload := &builderApi.VersionedSubmitBlindedBlockResponse{
			Version: spec.DataVersionCapella,
		}
		require.True(t, getPayloadResponseIsEmpty(payload))
	})

	t.Run("Nil block hash for capella payload response", func(t *testing.T) {
		payload := &builderApi.VersionedSubmitBlindedBlockResponse{
			Version: spec.DataVersionCapella,
			Capella: &capella.ExecutionPayload{
				BlockHash: nilHash,
			},
		}
		require.True(t, getPayloadResponseIsEmpty(payload))
	})

	t.Run("Empty deneb payload response", func(t *testing.T) {
		payload := &builderApi.VersionedSubmitBlindedBlockResponse{
			Version: spec.DataVersionDeneb,
		}
		require.True(t, getPayloadResponseIsEmpty(payload))
	})

	t.Run("Empty deneb execution payload", func(t *testing.T) {
		payload := &builderApi.VersionedSubmitBlindedBlockResponse{
			Version: spec.DataVersionDeneb,
			Deneb: &builderApiDeneb.ExecutionPayloadAndBlobsBundle{
				BlobsBundle: &builderApiDeneb.BlobsBundle{
					Blobs:       make([]deneb.Blob, 0),
					Commitments: make([]deneb.KZGCommitment, 0),
					Proofs:      make([]deneb.KZGProof, 0),
				},
			},
		}
		require.True(t, getPayloadResponseIsEmpty(payload))
	})

	t.Run("Empty deneb blobs bundle", func(t *testing.T) {
		payload := &builderApi.VersionedSubmitBlindedBlockResponse{
			Version: spec.DataVersionDeneb,
			Deneb: &builderApiDeneb.ExecutionPayloadAndBlobsBundle{
				ExecutionPayload: &deneb.ExecutionPayload{
					BlockHash: phase0.Hash32{0x1},
				},
			},
		}
		require.True(t, getPayloadResponseIsEmpty(payload))
	})

	t.Run("Nil block hash for deneb payload response", func(t *testing.T) {
		payload := &builderApi.VersionedSubmitBlindedBlockResponse{
			Version: spec.DataVersionDeneb,
			Deneb: &builderApiDeneb.ExecutionPayloadAndBlobsBundle{
				ExecutionPayload: &deneb.ExecutionPayload{
					BlockHash: nilHash,
				},
			},
		}
		require.True(t, getPayloadResponseIsEmpty(payload))
	})

	t.Run("Unsupported payload version", func(t *testing.T) {
		payload := &builderApi.VersionedSubmitBlindedBlockResponse{
			Version: spec.DataVersionBellatrix,
		}
		require.True(t, getPayloadResponseIsEmpty(payload))
	})
}
