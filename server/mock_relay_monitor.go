package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/flashbots/go-boost-utils/types"
	"github.com/gorilla/mux"
	"github.com/ralexstokes/relay-monitor/pkg/analysis"
	"github.com/ralexstokes/relay-monitor/pkg/api"
)

// mockRelayMonitor is used to fake a relay monitor's behaviour.
// You can override each of its handler by setting the instance's HandlerOverride_METHOD_TO_OVERRIDE to your own
// handler.
type mockRelayMonitor struct {
	// Used to panic if impossible error happens
	t *testing.T

	// Used to count each Request made to the relay, either if it fails or not, for each method
	mu           sync.Mutex
	requestCount map[string]int

	// Overriders
	handlerOverrideGetFaults func(w http.ResponseWriter, req *http.Request)

	// Default responses placeholders, used if overrider does not exist
	GetFaultsResponse *api.FaultsResponse

	// Server section
	Server *httptest.Server
}

// newMockRelayMonitor creates a mocked relay monitor which implements its API
// see https://github.com/ralexstokes/relay-monitor for the API spec
func newMockRelayMonitor(t *testing.T) *mockRelayMonitor {
	relayMonitor := &mockRelayMonitor{
		t:            t,
		requestCount: make(map[string]int),
	}
	relayMonitor.Server = httptest.NewServer(relayMonitor.getRouter())
	return relayMonitor
}

// newTestMiddleware creates a middleware which increases the Request counter and creates a fake delay for the response
func (m *mockRelayMonitor) newTestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// Request counter
			m.mu.Lock()
			url := r.URL.EscapedPath()
			m.requestCount[url]++
			m.mu.Unlock()

			next.ServeHTTP(w, r)
		},
	)
}

func (m *mockRelayMonitor) getRouter() http.Handler {
	// Create router.
	r := mux.NewRouter()

	// Register handlers
	r.HandleFunc(pathRegisterValidator, m.handleRegisterValidator).Methods(http.MethodPost)
	r.HandleFunc(pathAuctionTranscript, m.handleTranscript).Methods(http.MethodPost)
	r.HandleFunc(pathFault, m.handleGetFaults).Methods(http.MethodGet)

	return m.newTestMiddleware(r)
}

// GetRequestCount returns the number of Request made to a specific URL
func (m *mockRelayMonitor) GetRequestCount(path string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requestCount[path]
}

// By default, handleRegisterValidator returns a 200 OK status
func (m *mockRelayMonitor) handleRegisterValidator(w http.ResponseWriter, req *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	payload := []types.SignedValidatorRegistration{}
	if err := DecodeJSON(req.Body, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// By default, handleTranscript returns a 200 OK status
func (m *mockRelayMonitor) handleTranscript(w http.ResponseWriter, req *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	payload := AuctionTranscript{}
	if err := DecodeJSON(req.Body, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// MakeGetFaultsResponse is used to create the default or a custom response to the getFaults method
func (m *mockRelayMonitor) MakeGetFaultsResponse(start, end uint64, faultMap analysis.FaultRecord) *api.FaultsResponse {
	return &api.FaultsResponse{
		Span: api.Span{
			Start: start,
			End:   end,
		},
		FaultRecord: faultMap,
	}
}

// By default, handleGetFaults returns the default fault response unless overriden
func (m *mockRelayMonitor) handleGetFaults(w http.ResponseWriter, req *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Try to override default behaviour if customer handler is specified
	if m.handlerOverrideGetFaults != nil {
		m.handlerOverrideGetFaults(w, req)
		return
	}

	// return 200 response by default
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Build default response
	faultsRecord := make(analysis.FaultRecord)
	faultsRecord[_HexToPubkey(mockRelayPublicKeyHex)] = &analysis.Faults{
		TotalBids:                1153,
		MalformedBids:            0,
		ConsensusInvalidBids:     1,
		PaymentInvalidBids:       12,
		IgnoredPreferencesBids:   5,
		MalformedPayloads:        0,
		ConsensusInvalidPayloads: 1,
		UnavailablePayloads:      10,
	}
	response := m.MakeGetFaultsResponse(100, 200, faultsRecord)

	if m.GetFaultsResponse != nil {
		response = m.GetFaultsResponse
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
