package testing

import (
	"encoding/json"
	"fmt"
	"github.com/flashbots/go-boost-utils/bls"
	"github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/mev-boost/server"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// MockRelay is used to fake a relay's behavior.
// You can override each of its handler by setting the instance's HandlerOverride_METHOD_TO_OVERRIDE to your own
// handler.
type MockRelay struct {
	// Used to panic if impossible error happens
	t *testing.T

	// KeyPair used to sign messages
	secretKey *bls.SecretKey
	publicKey *bls.PublicKey

	// Used to count each request made to the relay, either if it fails or not, for each method
	mu           sync.Mutex
	requestCount map[string]int

	// Overriders
	HandlerOverrideRegisterValidator func(w http.ResponseWriter, req *http.Request)
	HandlerOverrideGetHeader         func(w http.ResponseWriter, req *http.Request)
	HandlerOverrideGetPayload        func(w http.ResponseWriter, req *http.Request)

	// Default responses placeholders, used if overrider does not exist
	GetHeaderResponse  *types.GetHeaderResponse
	GetPayloadResponse *types.GetPayloadResponse

	// Server section
	Server        *httptest.Server
	ResponseDelay time.Duration
}

// NewMockRelay creates a mocked relay which implements the server.BoostBackend interface
// A secret key must be provided to sign default and custom response messages
func NewMockRelay(t *testing.T, secretKey *bls.SecretKey) *MockRelay {
	publicKey := bls.PublicKeyFromSecretKey(secretKey)
	relay := &MockRelay{t: t, secretKey: secretKey, publicKey: publicKey, requestCount: make(map[string]int)}

	// Initialize server
	relay.Server = httptest.NewServer(relay.getRouter())

	return relay
}

// newTestMiddleware creates a middleware which increases the request counter and creates a fake delay for the response
func (m *MockRelay) newTestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// Request counter
			m.mu.Lock()
			url := r.URL.EscapedPath()
			m.requestCount[url]++
			m.mu.Unlock()

			// Artificial Delay
			if m.ResponseDelay > 0 {
				time.Sleep(m.ResponseDelay)
			}

			next.ServeHTTP(w, r)
		},
	)
}

// getRouter registers all methods from the backend, apply the test middleware a,nd return the configured router
func (m *MockRelay) getRouter() http.Handler {
	// Create router.
	r := mux.NewRouter()

	// Register handlers
	r.HandleFunc("/", m.handleRoot).Methods(http.MethodGet)
	r.HandleFunc(server.PathStatus, m.handleStatus).Methods(http.MethodGet)
	r.HandleFunc(server.PathRegisterValidator, m.handleRegisterValidator).Methods(http.MethodPost)
	r.HandleFunc(server.PathGetHeader, m.handleGetHeader).Methods(http.MethodGet)
	r.HandleFunc(server.PathGetPayload, m.handleGetPayload).Methods(http.MethodPost)

	return m.newTestMiddleware(r)
}

// getRequestCount returns the number of request made to a specific URL
func (m *MockRelay) getRequestCount(path string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requestCount[path]
}

// By default, handleRoot returns the relay's status
func (m *MockRelay) handleRoot(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{}`)
}

// By default, handleStatus returns the relay's status as http.StatusOK
func (m *MockRelay) handleStatus(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{}`)
}

// By default, handleRegisterValidator returns a default types.SignedValidatorRegistration
func (m *MockRelay) handleRegisterValidator(w http.ResponseWriter, req *http.Request) {
	if m.HandlerOverrideRegisterValidator != nil {
		m.HandlerOverrideRegisterValidator(w, req)
		return
	}

	payload := []types.SignedValidatorRegistration{}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// MakeGetHeaderResponse is used to create the default or can be used to create a custom response to the getHeader
// method
func (m *MockRelay) MakeGetHeaderResponse(value uint64, hash, publicKey string) *types.GetHeaderResponse {
	// Fill the payload with custom values.
	message := &types.BuilderBid{
		Header: &types.ExecutionPayloadHeader{
			BlockHash: _HexToHash(hash),
		},
		Value:  types.IntToU256(value),
		Pubkey: _HexToPubkey(publicKey),
	}

	// Sign the message.
	signature, err := types.SignMessage(message, types.DomainBuilder, m.secretKey)
	require.NoError(m.t, err)

	return &types.GetHeaderResponse{
		Version: "bellatrix",
		Data: &types.SignedBuilderBid{
			Message:   message,
			Signature: signature,
		},
	}
}

// handleGetHeader handles incoming requests to server.PathGetHeader
func (m *MockRelay) handleGetHeader(w http.ResponseWriter, req *http.Request) {
	// Try to override default behavior is custom handler is specified.
	if m.HandlerOverrideGetHeader != nil {
		m.HandlerOverrideGetHeader(w, req)
		return
	}

	// By default, everything will be ok.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Build the default response.
	response := m.MakeGetHeaderResponse(
		12345,
		"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
		"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
	)
	if m.GetHeaderResponse != nil {
		response = m.GetHeaderResponse
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// MakeGetPayloadResponse is used to create the default or can be used to create a custom response to the getPayload
// method
func (m *MockRelay) MakeGetPayloadResponse(parentHash, blockHash, feeRecipient string, blockNumber uint64) *types.GetPayloadResponse {
	return &types.GetPayloadResponse{
		Version: "bellatrix",
		Data: &types.ExecutionPayload{
			ParentHash:   _HexToHash(parentHash),
			BlockHash:    _HexToHash(blockHash),
			BlockNumber:  blockNumber,
			FeeRecipient: _HexToAddress(feeRecipient),
		},
	}
}

// handleGetPayload handles incoming requests to server.PathGetPayload
func (m *MockRelay) handleGetPayload(w http.ResponseWriter, req *http.Request) {
	// Try to override default behavior is custom handler is specified.
	if m.HandlerOverrideGetPayload != nil {
		m.HandlerOverrideGetPayload(w, req)
		return
	}

	// By default, everything will be ok.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Build the default response.
	response := m.MakeGetPayloadResponse(
		"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7",
		"0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab1",
		"0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941",
		12345,
	)
	if m.GetPayloadResponse != nil {
		response = m.GetPayloadResponse
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
