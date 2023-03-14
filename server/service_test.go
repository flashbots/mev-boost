package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/attestantio/go-builder-client/api"
	apiv1capella "github.com/attestantio/go-eth2-client/api/v1/capella"
	consensusspec "github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/mev-boost/config"
	"github.com/stretchr/testify/require"
)

type testBackend struct {
	boost  *BoostService
	relays []*mockRelay
}

// newTestBackend creates a new backend, initializes mock relays, registers them and return the instance
func newTestBackend(t *testing.T, numRelays int, relayTimeout time.Duration) *testBackend {
	t.Helper()
	backend := testBackend{
		relays: make([]*mockRelay, numRelays),
	}

	relayEntries := make([]RelayEntry, numRelays)
	for i := 0; i < numRelays; i++ {
		// Create a mock relay
		backend.relays[i] = newMockRelay(t)
		relayEntries[i] = backend.relays[i].RelayEntry
	}

	opts := BoostServiceOpts{
		Log:                      testLog,
		ListenAddr:               "localhost:12345",
		Relays:                   relayEntries,
		GenesisForkVersionHex:    "0x00000000",
		RelayCheck:               true,
		RelayMinBid:              types.IntToU256(12345),
		RequestTimeoutGetHeader:  relayTimeout,
		RequestTimeoutGetPayload: relayTimeout,
		RequestTimeoutRegVal:     relayTimeout,
		RequestMaxRetries:        5,
	}
	service, err := NewBoostService(opts)
	require.NoError(t, err)

	backend.boost = service
	return &backend
}

func (be *testBackend) request(t *testing.T, method, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	var err error

	if payload == nil {
		req, err = http.NewRequest(method, path, bytes.NewReader(nil))
	} else {
		payloadBytes, err2 := json.Marshal(payload)
		require.NoError(t, err2)
		req, err = http.NewRequest(method, path, bytes.NewReader(payloadBytes))
	}

	require.NoError(t, err)
	rr := httptest.NewRecorder()
	be.boost.getRouter().ServeHTTP(rr, req)
	return rr
}

func blindedBlockToExecutionPayloadBellatrix(signedBlindedBeaconBlock *types.SignedBlindedBeaconBlock) *types.ExecutionPayload {
	header := signedBlindedBeaconBlock.Message.Body.ExecutionPayloadHeader
	return &types.ExecutionPayload{
		ParentHash:    header.ParentHash,
		FeeRecipient:  header.FeeRecipient,
		StateRoot:     header.StateRoot,
		ReceiptsRoot:  header.ReceiptsRoot,
		LogsBloom:     header.LogsBloom,
		Random:        header.Random,
		BlockNumber:   header.BlockNumber,
		GasLimit:      header.GasLimit,
		GasUsed:       header.GasUsed,
		Timestamp:     header.Timestamp,
		ExtraData:     header.ExtraData,
		BaseFeePerGas: header.BaseFeePerGas,
		BlockHash:     header.BlockHash,
	}
}

func blindedBlockToExecutionPayloadCapella(signedBlindedBeaconBlock *apiv1capella.SignedBlindedBeaconBlock) *capella.ExecutionPayload {
	header := signedBlindedBeaconBlock.Message.Body.ExecutionPayloadHeader
	return &capella.ExecutionPayload{
		ParentHash:    header.ParentHash,
		FeeRecipient:  header.FeeRecipient,
		StateRoot:     header.StateRoot,
		ReceiptsRoot:  header.ReceiptsRoot,
		LogsBloom:     header.LogsBloom,
		PrevRandao:    header.PrevRandao,
		BlockNumber:   header.BlockNumber,
		GasLimit:      header.GasLimit,
		GasUsed:       header.GasUsed,
		Timestamp:     header.Timestamp,
		ExtraData:     header.ExtraData,
		BaseFeePerGas: header.BaseFeePerGas,
		BlockHash:     header.BlockHash,
		Transactions:  make([]bellatrix.Transaction, 0),
		Withdrawals:   make([]*capella.Withdrawal, 0),
	}
}

func TestNewBoostServiceErrors(t *testing.T) {
	t.Run("errors when no relays", func(t *testing.T) {
		_, err := NewBoostService(BoostServiceOpts{testLog, ":123", []RelayEntry{}, []*url.URL{}, "0x00000000", true, types.IntToU256(0), time.Second, time.Second, time.Second, 1})
		require.Error(t, err)
	})
}

func TestWebserver(t *testing.T) {
	t.Run("errors when webserver is already existing", func(t *testing.T) {
		backend := newTestBackend(t, 1, time.Second)
		backend.boost.srv = &http.Server{}
		err := backend.boost.StartHTTPServer()
		require.Error(t, err)
	})

	t.Run("webserver error on invalid listenAddr", func(t *testing.T) {
		backend := newTestBackend(t, 1, time.Second)
		backend.boost.listenAddr = "localhost:876543"
		err := backend.boost.StartHTTPServer()
		require.Error(t, err)
	})

	// t.Run("webserver starts normally", func(t *testing.T) {
	// 	backend := newTestBackend(t, 1, time.Second)
	// 	go func() {
	// 		err := backend.boost.StartHTTPServer()
	// 		require.NoError(t, err)
	// 	}()
	// 	time.Sleep(time.Millisecond * 100)
	// 	backend.boost.srv.Close()
	// })
}

func TestWebserverRootHandler(t *testing.T) {
	backend := newTestBackend(t, 1, time.Second)

	// Check root handler
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	backend.boost.getRouter().ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "{}\n", rr.Body.String())
}

func TestWebserverMaxHeaderSize(t *testing.T) {
	backend := newTestBackend(t, 1, time.Second)
	addr := "localhost:1234"
	backend.boost.listenAddr = addr
	go func() {
		err := backend.boost.StartHTTPServer()
		require.NoError(t, err)
	}()
	time.Sleep(time.Millisecond * 100)
	path := "http://" + addr + "?" + strings.Repeat("abc", 4000) // path with characters of size over 4kb
	code, err := SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, path, "test", nil, nil)
	require.Error(t, err)
	require.Equal(t, http.StatusRequestHeaderFieldsTooLarge, code)
	backend.boost.srv.Close()
}

func TestStatus(t *testing.T) {
	t.Run("At least one relay is available", func(t *testing.T) {
		backend := newTestBackend(t, 1, time.Second)
		time.Sleep(time.Millisecond * 20)
		path := "/eth/v1/builder/status"
		rr := backend.request(t, http.MethodGet, path, nil)

		require.Equal(t, http.StatusOK, rr.Code)
		require.True(t, len(rr.Header().Get("X-MEVBoost-Version")) > 0)
		require.Equal(t, config.ForkVersion, rr.Header().Get("X-MEVBoost-ForkVersion"))
		require.Equal(t, 1, backend.relays[0].GetRequestCount(path))
	})

	t.Run("No relays available", func(t *testing.T) {
		backend := newTestBackend(t, 1, time.Second)
		backend.relays[0].Server.Close() // makes the relay unavailable

		path := "/eth/v1/builder/status"
		rr := backend.request(t, http.MethodGet, path, nil)

		require.Equal(t, http.StatusServiceUnavailable, rr.Code)
		require.True(t, len(rr.Header().Get("X-MEVBoost-Version")) > 0)
		require.Equal(t, config.ForkVersion, rr.Header().Get("X-MEVBoost-ForkVersion"))
		require.Equal(t, 0, backend.relays[0].GetRequestCount(path))
	})
}

func TestRegisterValidator(t *testing.T) {
	path := "/eth/v1/builder/validators"
	reg := types.SignedValidatorRegistration{
		Message: &types.RegisterValidatorRequestMessage{
			FeeRecipient: _HexToAddress("0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941"),
			Timestamp:    1234356,
			Pubkey: _HexToPubkey(
				"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249"),
		},
		Signature: _HexToSignature(
			"0x81510b571e22f89d1697545aac01c9ad0c1e7a3e778b3078bef524efae14990e58a6e960a152abd49de2e18d7fd3081c15d5c25867ccfad3d47beef6b39ac24b6b9fbf2cfa91c88f67aff750438a6841ec9e4a06a94ae41410c4f97b75ab284c"),
	}
	payload := []types.SignedValidatorRegistration{reg}

	t.Run("Normal function", func(t *testing.T) {
		backend := newTestBackend(t, 1, time.Second)
		rr := backend.request(t, http.MethodPost, path, payload)
		require.Equal(t, http.StatusOK, rr.Code)
		require.Equal(t, 1, backend.relays[0].GetRequestCount(path))
	})

	t.Run("Relay error response", func(t *testing.T) {
		backend := newTestBackend(t, 2, time.Second)

		backend.relays[0].ResponseDelay = 5 * time.Millisecond
		backend.relays[1].ResponseDelay = 5 * time.Millisecond

		rr := backend.request(t, http.MethodPost, path, payload)
		require.Equal(t, http.StatusOK, rr.Code)
		require.Equal(t, 1, backend.relays[0].GetRequestCount(path))
		require.Equal(t, 1, backend.relays[1].GetRequestCount(path))

		// Now make one relay return an error
		backend.relays[0].overrideHandleRegisterValidator(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		})
		rr = backend.request(t, http.MethodPost, path, payload)
		require.Equal(t, http.StatusOK, rr.Code)
		require.Equal(t, 2, backend.relays[0].GetRequestCount(path))
		require.Equal(t, 2, backend.relays[1].GetRequestCount(path))

		// Now make both relays return an error - which should cause the request to fail
		backend.relays[1].overrideHandleRegisterValidator(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		})
		rr = backend.request(t, http.MethodPost, path, payload)
		require.Equal(t, `{"code":502,"message":"no successful relay response"}`+"\n", rr.Body.String())
		require.Equal(t, http.StatusBadGateway, rr.Code)
		require.Equal(t, 3, backend.relays[0].GetRequestCount(path))
		require.Equal(t, 3, backend.relays[1].GetRequestCount(path))
	})

	t.Run("mev-boost relay timeout works with slow relay", func(t *testing.T) {
		backend := newTestBackend(t, 1, 150*time.Millisecond) // 10ms max
		rr := backend.request(t, http.MethodPost, path, payload)
		require.Equal(t, http.StatusOK, rr.Code)

		// Now make the relay return slowly, mev-boost should return an error
		backend.relays[0].ResponseDelay = 180 * time.Millisecond
		rr = backend.request(t, http.MethodPost, path, payload)
		require.Equal(t, `{"code":502,"message":"no successful relay response"}`+"\n", rr.Body.String())
		require.Equal(t, http.StatusBadGateway, rr.Code)
		require.Equal(t, 2, backend.relays[0].GetRequestCount(path))
	})
}

func getHeaderPath(slot uint64, parentHash types.Hash, pubkey types.PublicKey) string {
	return fmt.Sprintf("/eth/v1/builder/header/%d/%s/%s", slot, parentHash.String(), pubkey.String())
}

func TestGetHeader(t *testing.T) {
	hash := _HexToHash("0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7")
	pubkey := _HexToPubkey(
		"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249")
	path := getHeaderPath(1, hash, pubkey)
	require.Equal(t, "/eth/v1/builder/header/1/0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7/0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249", path)

	t.Run("Okay response from relay", func(t *testing.T) {
		backend := newTestBackend(t, 1, time.Second)
		rr := backend.request(t, http.MethodGet, path, nil)
		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
		require.Equal(t, 1, backend.relays[0].GetRequestCount(path))
	})

	t.Run("Bad response from relays", func(t *testing.T) {
		backend := newTestBackend(t, 2, time.Second)
		resp := backend.relays[0].MakeGetHeaderResponse(
			12345,
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
			consensusspec.DataVersionBellatrix,
		)
		resp.Bellatrix.Data.Message.Header.BlockHash = nilHash

		// 1/2 failing responses are okay
		backend.relays[0].GetHeaderResponse = resp
		rr := backend.request(t, http.MethodGet, path, nil)
		require.Equal(t, 1, backend.relays[0].GetRequestCount(path))
		require.Equal(t, 1, backend.relays[1].GetRequestCount(path))
		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

		// 2/2 failing responses are okay
		backend.relays[1].GetHeaderResponse = resp
		rr = backend.request(t, http.MethodGet, path, nil)
		require.Equal(t, 2, backend.relays[0].GetRequestCount(path))
		require.Equal(t, 2, backend.relays[1].GetRequestCount(path))
		require.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("Invalid relay public key", func(t *testing.T) {
		backend := newTestBackend(t, 1, time.Second)

		backend.relays[0].GetHeaderResponse = backend.relays[0].MakeGetHeaderResponse(
			12345,
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
			consensusspec.DataVersionBellatrix,
		)

		// Simulate a different public key registered to mev-boost
		pk := types.PublicKey{}
		backend.boost.relays[0].PublicKey = pk

		rr := backend.request(t, http.MethodGet, path, nil)
		require.Equal(t, 1, backend.relays[0].GetRequestCount(path))

		// Request should have no content
		require.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("Invalid relay signature", func(t *testing.T) {
		backend := newTestBackend(t, 1, time.Second)

		backend.relays[0].GetHeaderResponse = backend.relays[0].MakeGetHeaderResponse(
			12345,
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
			consensusspec.DataVersionBellatrix,
		)

		// Scramble the signature
		backend.relays[0].GetHeaderResponse.Bellatrix.Data.Signature = types.Signature{}

		rr := backend.request(t, http.MethodGet, path, nil)
		require.Equal(t, 1, backend.relays[0].GetRequestCount(path))

		// Request should have no content
		require.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("Invalid slot number", func(t *testing.T) {
		// Number larger than uint64 creates parsing error
		slot := fmt.Sprintf("%d0", uint64(math.MaxUint64))
		invalidSlotPath := fmt.Sprintf("/eth/v1/builder/header/%s/%s/%s", slot, hash.String(), pubkey.String())

		backend := newTestBackend(t, 1, time.Second)
		rr := backend.request(t, http.MethodGet, invalidSlotPath, nil)
		require.Equal(t, `{"code":400,"message":"invalid slot"}`+"\n", rr.Body.String())
		require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
		require.Equal(t, 0, backend.relays[0].GetRequestCount(path))
	})

	t.Run("Invalid pubkey length", func(t *testing.T) {
		invalidPubkeyPath := fmt.Sprintf("/eth/v1/builder/header/%d/%s/%s", 1, hash.String(), "0x1")

		backend := newTestBackend(t, 1, time.Second)
		rr := backend.request(t, http.MethodGet, invalidPubkeyPath, nil)
		require.Equal(t, `{"code":400,"message":"invalid pubkey"}`+"\n", rr.Body.String())
		require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
		require.Equal(t, 0, backend.relays[0].GetRequestCount(path))
	})

	t.Run("Invalid hash length", func(t *testing.T) {
		invalidSlotPath := fmt.Sprintf("/eth/v1/builder/header/%d/%s/%s", 1, "0x1", pubkey.String())

		backend := newTestBackend(t, 1, time.Second)
		rr := backend.request(t, http.MethodGet, invalidSlotPath, nil)
		require.Equal(t, `{"code":400,"message":"invalid hash"}`+"\n", rr.Body.String())
		require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
		require.Equal(t, 0, backend.relays[0].GetRequestCount(path))
	})

	t.Run("Invalid parent hash", func(t *testing.T) {
		backend := newTestBackend(t, 1, time.Second)

		invalidParentHashPath := getHeaderPath(1, types.Hash{}, pubkey)
		rr := backend.request(t, http.MethodGet, invalidParentHashPath, nil)
		require.Equal(t, http.StatusNoContent, rr.Code)
		require.Equal(t, 0, backend.relays[0].GetRequestCount(path))
	})
}

func TestGetHeaderBids(t *testing.T) {
	hash := _HexToHash("0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7")
	pubkey := _HexToPubkey(
		"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249")
	path := getHeaderPath(2, hash, pubkey)
	require.Equal(t, "/eth/v1/builder/header/2/0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7/0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249", path)

	t.Run("Use header with highest value", func(t *testing.T) {
		// Create backend and register 3 relays.
		backend := newTestBackend(t, 3, time.Second)

		// First relay will return signed response with value 12345.
		backend.relays[0].GetHeaderResponse = backend.relays[0].MakeGetHeaderResponse(
			12345,
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
			consensusspec.DataVersionBellatrix,
		)

		// First relay will return signed response with value 12347.
		backend.relays[1].GetHeaderResponse = backend.relays[1].MakeGetHeaderResponse(
			12347,
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
			consensusspec.DataVersionBellatrix,
		)

		// First relay will return signed response with value 12346.
		backend.relays[2].GetHeaderResponse = backend.relays[2].MakeGetHeaderResponse(
			12346,
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
			consensusspec.DataVersionBellatrix,
		)

		// Run the request.
		rr := backend.request(t, http.MethodGet, path, nil)

		// Each relay must have received the request.
		require.Equal(t, 1, backend.relays[0].GetRequestCount(path))
		require.Equal(t, 1, backend.relays[1].GetRequestCount(path))
		require.Equal(t, 1, backend.relays[2].GetRequestCount(path))

		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

		// Highest value should be 12347, i.e. second relay.
		resp := new(GetHeaderResponse)
		err := json.Unmarshal(rr.Body.Bytes(), resp)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(12347), resp.Value())
	})

	t.Run("Use header with lowest blockhash if same value", func(t *testing.T) {
		// Create backend and register 3 relays.
		backend := newTestBackend(t, 3, time.Second)

		backend.relays[0].GetHeaderResponse = backend.relays[0].MakeGetHeaderResponse(
			12345,
			"0xa38385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
			consensusspec.DataVersionBellatrix,
		)

		backend.relays[1].GetHeaderResponse = backend.relays[1].MakeGetHeaderResponse(
			12345,
			"0xa18385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
			consensusspec.DataVersionBellatrix,
		)

		backend.relays[2].GetHeaderResponse = backend.relays[2].MakeGetHeaderResponse(
			12345,
			"0xa28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
			consensusspec.DataVersionBellatrix,
		)

		// Run the request.
		rr := backend.request(t, http.MethodGet, path, nil)

		// Each relay must have received the request.
		require.Equal(t, 1, backend.relays[0].GetRequestCount(path))
		require.Equal(t, 1, backend.relays[1].GetRequestCount(path))
		require.Equal(t, 1, backend.relays[2].GetRequestCount(path))

		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

		// Highest value should be 12347, i.e. second relay.
		resp := new(GetHeaderResponse)

		err := json.Unmarshal(rr.Body.Bytes(), resp)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(12345), resp.Value())
		require.Equal(t, "0xa18385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7", resp.BlockHash())
	})

	t.Run("Respect minimum bid cutoff", func(t *testing.T) {
		// Create backend and register relay.
		backend := newTestBackend(t, 1, time.Second)

		// Relay will return signed response with value 12344.
		backend.relays[0].GetHeaderResponse = backend.relays[0].MakeGetHeaderResponse(
			12344,
			"0xa28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
			consensusspec.DataVersionBellatrix,
		)

		// Run the request.
		rr := backend.request(t, http.MethodGet, path, nil)

		// Each relay must have received the request.
		require.Equal(t, 1, backend.relays[0].GetRequestCount(path))

		// Request should have no content (min bid is 12345)
		require.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("Allow bids which meet minimum bid cutoff", func(t *testing.T) {
		// Create backend and register relay.
		backend := newTestBackend(t, 1, time.Second)

		// First relay will return signed response with value 12345.
		backend.relays[0].GetHeaderResponse = backend.relays[0].MakeGetHeaderResponse(
			12345,
			"0xa28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
			consensusspec.DataVersionBellatrix,
		)

		// Run the request.
		rr := backend.request(t, http.MethodGet, path, nil)

		// Each relay must have received the request.
		require.Equal(t, 1, backend.relays[0].GetRequestCount(path))

		// Value should be 12345 (min bid is 12345)
		resp := new(types.GetHeaderResponse)
		err := json.Unmarshal(rr.Body.Bytes(), resp)
		require.NoError(t, err)
		require.Equal(t, types.IntToU256(12345), resp.Data.Message.Value)
	})
}

func TestGetHeaderCapellaBids(t *testing.T) {
	hash := _HexToHash("0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7")
	pubkey := _HexToPubkey(
		"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249")
	path := getHeaderPath(1, hash, pubkey)
	require.Equal(t, "/eth/v1/builder/header/1/0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7/0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249", path)

	t.Run("Use header with highest value", func(t *testing.T) {
		// Create backend and register 3 relays.
		backend := newTestBackend(t, 3, time.Second)

		// First relay will return signed response with value 12345.
		backend.relays[0].GetHeaderResponse = backend.relays[0].MakeGetHeaderResponse(
			12345,
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
			consensusspec.DataVersionCapella,
		)

		// First relay will return signed response with value 12347.
		backend.relays[1].GetHeaderResponse = backend.relays[1].MakeGetHeaderResponse(
			12347,
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
			consensusspec.DataVersionCapella,
		)

		// First relay will return signed response with value 12346.
		backend.relays[2].GetHeaderResponse = backend.relays[2].MakeGetHeaderResponse(
			12346,
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
			consensusspec.DataVersionCapella,
		)

		// Run the request.
		rr := backend.request(t, http.MethodGet, path, nil)

		// Each relay must have received the request.
		require.Equal(t, 1, backend.relays[0].GetRequestCount(path))
		require.Equal(t, 1, backend.relays[1].GetRequestCount(path))
		require.Equal(t, 1, backend.relays[2].GetRequestCount(path))

		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

		// Highest value should be 12347, i.e. second relay.
		resp := new(GetHeaderResponse)
		err := json.Unmarshal(rr.Body.Bytes(), resp)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(12347), resp.Value())
	})

	t.Run("Use header with lowest blockhash if same value", func(t *testing.T) {
		// Create backend and register 3 relays.
		backend := newTestBackend(t, 3, time.Second)

		backend.relays[0].GetHeaderResponse = backend.relays[0].MakeGetHeaderResponse(
			12345,
			"0xa38385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
			consensusspec.DataVersionCapella,
		)

		backend.relays[1].GetHeaderResponse = backend.relays[1].MakeGetHeaderResponse(
			12345,
			"0xa18385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
			consensusspec.DataVersionCapella,
		)

		backend.relays[2].GetHeaderResponse = backend.relays[2].MakeGetHeaderResponse(
			12345,
			"0xa28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
			consensusspec.DataVersionCapella,
		)

		// Run the request.
		rr := backend.request(t, http.MethodGet, path, nil)

		// Each relay must have received the request.
		require.Equal(t, 1, backend.relays[0].GetRequestCount(path))
		require.Equal(t, 1, backend.relays[1].GetRequestCount(path))
		require.Equal(t, 1, backend.relays[2].GetRequestCount(path))

		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

		// Highest value should be 12347, i.e. second relay.
		resp := new(GetHeaderResponse)

		err := json.Unmarshal(rr.Body.Bytes(), resp)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(12345), resp.Value())
		require.Equal(t, "0xa18385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7", resp.BlockHash())
	})

	t.Run("Respect minimum bid cutoff", func(t *testing.T) {
		// Create backend and register relay.
		backend := newTestBackend(t, 1, time.Second)

		// Relay will return signed response with value 12344.
		backend.relays[0].GetHeaderResponse = backend.relays[0].MakeGetHeaderResponse(
			12344,
			"0xa28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
			consensusspec.DataVersionCapella,
		)

		// Run the request.
		rr := backend.request(t, http.MethodGet, path, nil)

		// Each relay must have received the request.
		require.Equal(t, 1, backend.relays[0].GetRequestCount(path))

		// Request should have no content (min bid is 12345)
		require.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("Allow bids which meet minimum bid cutoff", func(t *testing.T) {
		// Create backend and register relay.
		backend := newTestBackend(t, 1, time.Second)

		// First relay will return signed response with value 12345.
		backend.relays[0].GetHeaderResponse = backend.relays[0].MakeGetHeaderResponse(
			12345,
			"0xa28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
			consensusspec.DataVersionCapella,
		)

		// Run the request.
		rr := backend.request(t, http.MethodGet, path, nil)

		// Each relay must have received the request.
		require.Equal(t, 1, backend.relays[0].GetRequestCount(path))

		// Value should be 12345 (min bid is 12345)
		resp := new(types.GetHeaderResponse)
		err := json.Unmarshal(rr.Body.Bytes(), resp)
		require.NoError(t, err)
		require.Equal(t, types.IntToU256(12345), resp.Data.Message.Value)
	})

	t.Run("Invalid relay signature", func(t *testing.T) {
		backend := newTestBackend(t, 1, time.Second)

		backend.relays[0].GetHeaderResponse = backend.relays[0].MakeGetHeaderResponse(
			12345,
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
			consensusspec.DataVersionCapella,
		)

		// Scramble the signature
		backend.relays[0].GetHeaderResponse.Capella.Capella.Signature = phase0.BLSSignature{}

		rr := backend.request(t, http.MethodGet, path, nil)
		require.Equal(t, 1, backend.relays[0].GetRequestCount(path))

		// Request should have no content
		require.Equal(t, http.StatusNoContent, rr.Code)
	})
}

func TestGetPayload(t *testing.T) {
	path := "/eth/v1/builder/blinded_blocks"

	payload := types.SignedBlindedBeaconBlock{
		Signature: _HexToSignature(
			"0x8c795f751f812eabbabdee85100a06730a9904a4b53eedaa7f546fe0e23cd75125e293c6b0d007aa68a9da4441929d16072668abb4323bb04ac81862907357e09271fe414147b3669509d91d8ffae2ec9c789a5fcd4519629b8f2c7de8d0cce9"),
		Message: &types.BlindedBeaconBlock{
			Slot:          1,
			ProposerIndex: 1,
			ParentRoot:    types.Root{0x01},
			StateRoot:     types.Root{0x02},
			Body: &types.BlindedBeaconBlockBody{
				RandaoReveal:  types.Signature{0xa1},
				Eth1Data:      &types.Eth1Data{},
				Graffiti:      types.Hash{0xa2},
				SyncAggregate: &types.SyncAggregate{},
				ExecutionPayloadHeader: &types.ExecutionPayloadHeader{
					ParentHash:   _HexToHash("0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7"),
					BlockHash:    _HexToHash("0x534809bd2b6832edff8d8ce4cb0e50068804fd1ef432c8362ad708a74fdc0e46"),
					BlockNumber:  12345,
					FeeRecipient: _HexToAddress("0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941"),
				},
			},
		},
	}

	t.Run("Okay response from relay", func(t *testing.T) {
		backend := newTestBackend(t, 1, time.Second)
		rr := backend.request(t, http.MethodPost, path, payload)
		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
		require.Equal(t, 1, backend.relays[0].GetRequestCount(path))

		resp := new(types.GetPayloadResponse)
		err := json.Unmarshal(rr.Body.Bytes(), resp)
		require.NoError(t, err)
		require.Equal(t, payload.Message.Body.ExecutionPayloadHeader.BlockHash, resp.Data.BlockHash)
	})

	t.Run("Bad response from relays", func(t *testing.T) {
		backend := newTestBackend(t, 2, time.Second)
		resp := new(types.GetPayloadResponse)

		// 1/2 failing responses are okay
		backend.relays[0].GetBellatrixPayloadResponse = resp
		rr := backend.request(t, http.MethodPost, path, payload)
		require.GreaterOrEqual(t, backend.relays[1].GetRequestCount(path)+backend.relays[0].GetRequestCount(path), 1)
		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

		// 2/2 failing responses are okay
		backend = newTestBackend(t, 2, time.Second)
		backend.relays[0].GetBellatrixPayloadResponse = resp
		backend.relays[1].GetBellatrixPayloadResponse = resp
		rr = backend.request(t, http.MethodPost, path, payload)
		require.Equal(t, 1, backend.relays[0].GetRequestCount(path))
		require.Equal(t, 1, backend.relays[1].GetRequestCount(path))
		require.Equal(t, `{"code":502,"message":"no successful relay response"}`+"\n", rr.Body.String())
		require.Equal(t, http.StatusBadGateway, rr.Code, rr.Body.String())
	})

	t.Run("Retries on error from relay", func(t *testing.T) {
		backend := newTestBackend(t, 1, 2*time.Second)

		count := 0
		backend.relays[0].handlerOverrideGetPayload = func(w http.ResponseWriter, r *http.Request) {
			if count > 0 {
				// success response on the second attempt
				backend.relays[0].defaultHandleGetPayload(w)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				_, err := w.Write([]byte(`{"code":500,"message":"internal server error"}`))
				require.NoError(t, err)
			}
			count++
		}
		rr := backend.request(t, http.MethodPost, path, payload)
		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	})

	t.Run("Error after max retries are reached", func(t *testing.T) {
		backend := newTestBackend(t, 1, 0)

		count := 0
		maxRetries := 5

		backend.relays[0].handlerOverrideGetPayload = func(w http.ResponseWriter, r *http.Request) {
			count++
			if count > maxRetries {
				// success response after max retry attempts
				backend.relays[0].defaultHandleGetPayload(w)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				_, err := w.Write([]byte(`{"code":500,"message":"internal server error"}`))
				require.NoError(t, err)
			}
		}
		rr := backend.request(t, http.MethodPost, path, payload)
		require.Equal(t, 5, backend.relays[0].GetRequestCount(path))
		require.Equal(t, `{"code":502,"message":"no successful relay response"}`+"\n", rr.Body.String())
		require.Equal(t, http.StatusBadGateway, rr.Code, rr.Body.String())
	})
}

func TestCheckRelays(t *testing.T) {
	t.Run("One relay is okay", func(t *testing.T) {
		backend := newTestBackend(t, 1, time.Second)
		numHealthyRelays := backend.boost.CheckRelays()
		require.Equal(t, 1, numHealthyRelays)
	})

	t.Run("One relay is down", func(t *testing.T) {
		backend := newTestBackend(t, 1, time.Second)
		backend.relays[0].Server.Close()

		numHealthyRelays := backend.boost.CheckRelays()
		require.Equal(t, 0, numHealthyRelays)
	})

	t.Run("One relays is up, one down", func(t *testing.T) {
		backend := newTestBackend(t, 2, time.Second)
		backend.relays[0].Server.Close()

		numHealthyRelays := backend.boost.CheckRelays()
		require.Equal(t, 1, numHealthyRelays)
	})

	t.Run("Should not follow redirects", func(t *testing.T) {
		backend := newTestBackend(t, 1, time.Second)
		redirectAddress := backend.relays[0].Server.URL
		backend.relays[0].Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, redirectAddress, http.StatusTemporaryRedirect)
		}))

		url, err := url.ParseRequestURI(backend.relays[0].Server.URL)
		require.NoError(t, err)
		backend.boost.relays[0].URL = url
		numHealthyRelays := backend.boost.CheckRelays()
		require.Equal(t, 0, numHealthyRelays)
	})
}

func TestEmptyTxRoot(t *testing.T) {
	transactions := types.Transactions{}
	txroot, _ := transactions.HashTreeRoot()
	txRootHex := fmt.Sprintf("0x%x", txroot)
	require.Equal(t, "0x7ffe241ea60187fdb0187bfa22de35d1f9bed7ab061d9401fd47e34a54fbede1", txRootHex)
}

func TestGetPayloadWithTestdata(t *testing.T) {
	path := "/eth/v1/builder/blinded_blocks"

	testPayloadsFiles := []string{
		"../testdata/kiln-signed-blinded-beacon-block-899730.json",
		"../testdata/signed-blinded-beacon-block-case0.json",
	}

	for _, fn := range testPayloadsFiles {
		t.Run(fn, func(t *testing.T) {
			jsonFile, err := os.Open(fn)
			require.NoError(t, err)
			defer jsonFile.Close()
			signedBlindedBeaconBlock := new(types.SignedBlindedBeaconBlock)
			require.NoError(t, DecodeJSON(jsonFile, &signedBlindedBeaconBlock))

			backend := newTestBackend(t, 1, time.Second)
			mockResp := types.GetPayloadResponse{
				Data: &types.ExecutionPayload{
					BlockHash: signedBlindedBeaconBlock.Message.Body.ExecutionPayloadHeader.BlockHash,
				},
			}
			backend.relays[0].GetBellatrixPayloadResponse = &mockResp

			rr := backend.request(t, http.MethodPost, path, signedBlindedBeaconBlock)
			require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
			require.Equal(t, 1, backend.relays[0].GetRequestCount(path))

			resp := new(types.GetPayloadResponse)
			err = json.Unmarshal(rr.Body.Bytes(), resp)
			require.NoError(t, err)
			require.Equal(t, signedBlindedBeaconBlock.Message.Body.ExecutionPayloadHeader.BlockHash, resp.Data.BlockHash)
		})
	}
}

func TestGetPayloadCapella(t *testing.T) {
	// Load the signed blinded beacon block used for getPayload
	jsonFile, err := os.Open("../testdata/signed-blinded-beacon-block-capella.json")
	require.NoError(t, err)
	defer jsonFile.Close()
	signedBlindedBeaconBlock := new(apiv1capella.SignedBlindedBeaconBlock)
	require.NoError(t, DecodeJSON(jsonFile, &signedBlindedBeaconBlock))

	backend := newTestBackend(t, 1, time.Second)

	// Prepare getPayload response
	backend.relays[0].GetCapellaPayloadResponse = &api.VersionedExecutionPayload{
		Version: consensusspec.DataVersionCapella,
		Capella: blindedBlockToExecutionPayloadCapella(signedBlindedBeaconBlock),
	}

	// call getPayload, ensure it's only called on relay 0 (origin of the bid)
	getPayloadPath := "/eth/v1/builder/blinded_blocks"
	rr := backend.request(t, http.MethodPost, getPayloadPath, signedBlindedBeaconBlock)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	require.Equal(t, 1, backend.relays[0].GetRequestCount(getPayloadPath))

	resp := new(api.VersionedExecutionPayload)
	err = json.Unmarshal(rr.Body.Bytes(), resp)
	require.NoError(t, err)
	require.Equal(t, signedBlindedBeaconBlock.Message.Body.ExecutionPayloadHeader.BlockHash, resp.Capella.BlockHash)
}

func TestGetPayloadToOriginRelayOnly(t *testing.T) {
	// Load the signed blinded beacon block used for getPayload
	jsonFile, err := os.Open("../testdata/kiln-signed-blinded-beacon-block-899730.json")
	require.NoError(t, err)
	defer jsonFile.Close()
	signedBlindedBeaconBlock := new(types.SignedBlindedBeaconBlock)
	require.NoError(t, DecodeJSON(jsonFile, &signedBlindedBeaconBlock))

	// Create a test backend with 2 relays
	backend := newTestBackend(t, 2, time.Second)

	// call getHeader, highest bid is returned by relay 0
	getHeaderPath := "/eth/v1/builder/header/899730/0xe8b9bd82aa0e957736c5a029903e53d581edf451e28ab274f4ba314c442e35a4/0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249"
	backend.relays[0].GetHeaderResponse = backend.relays[0].MakeGetHeaderResponse(
		12345,
		"0x373fb4e59dcb659b94bd58595c25345333426aa639f821567103e2eccf34d126",
		"0xe8b9bd82aa0e957736c5a029903e53d581edf451e28ab274f4ba314c442e35a4",
		"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
		consensusspec.DataVersionBellatrix,
	)
	rr := backend.request(t, http.MethodGet, getHeaderPath, nil)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	require.Equal(t, 1, backend.relays[0].GetRequestCount(getHeaderPath))
	require.Equal(t, 1, backend.relays[1].GetRequestCount(getHeaderPath))

	// Prepare getPayload response
	backend.relays[0].GetBellatrixPayloadResponse = &types.GetPayloadResponse{
		Data: blindedBlockToExecutionPayloadBellatrix(signedBlindedBeaconBlock),
	}

	// call getPayload, ensure it's only called on relay 0 (origin of the bid)
	getPayloadPath := "/eth/v1/builder/blinded_blocks"
	rr = backend.request(t, http.MethodPost, getPayloadPath, signedBlindedBeaconBlock)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	require.Equal(t, 1, backend.relays[0].GetRequestCount(getPayloadPath))
	require.Equal(t, 0, backend.relays[1].GetRequestCount(getPayloadPath))
}
