package testing

import (
	"bytes"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/go-boost-utils/bls"
	"github.com/flashbots/mev-boost/server"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type TestBackend struct {
	boost  *server.BoostService
	relays []*MockRelay
}

// NewTestBackend creates a new backend, initializes mock relays, registers them and return the instance
func NewTestBackend(t *testing.T, numRelays int, relayTimeout time.Duration) *TestBackend {
	backend := TestBackend{
		relays: make([]*MockRelay, numRelays),
	}

	relayEntries := make([]server.RelayEntry, numRelays)
	for i := 0; i < numRelays; i++ {
		// Generate private key for relay
		blsPrivateKey, blsPublicKey, err := bls.GenerateNewKeypair()
		require.NoError(t, err)

		// Create a mock relay
		backend.relays[i] = NewMockRelay(t, blsPrivateKey)

		// Create the server.RelayEntry used to identify the relay
		relayEntries[i], err = server.NewRelayEntry(backend.relays[i].Server.URL)
		require.NoError(t, err)

		// Hardcode relay's public key
		publicKeyString := hexutil.Encode(blsPublicKey.Compress())
		publicKey := _HexToPubkey(publicKeyString)
		relayEntries[i].PublicKey = publicKey
	}
	service, err := server.NewBoostService("localhost:12345", relayEntries, testLog, relayTimeout)
	require.NoError(t, err)

	backend.boost = service
	return &backend
}

func (be *TestBackend) request(t *testing.T, method string, path string, payload any) *httptest.ResponseRecorder {
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
	be.boost.GetRouter().ServeHTTP(rr, req)
	return rr
}
