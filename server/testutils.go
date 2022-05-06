package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/mev-boost/types"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var testLog = logrus.WithField("testing", true)

func _hexToBytes(hex string) []byte {
	bytes, err := hexutil.Decode(hex)
	if err != nil {
		panic(err)
	}
	return bytes
}

type testBackend struct {
	boost  *BoostService
	relays []*mockRelay
}

func newTestBackend(t *testing.T, numRelays int, relayTimeout time.Duration) *testBackend {
	resp := testBackend{
		relays: make([]*mockRelay, numRelays),
	}

	relayEntries := make([]types.RelayEntry, numRelays)
	for i := 0; i < numRelays; i++ {
		resp.relays[i] = newMockRelay()
		relayEntries[i].Address = resp.relays[i].Server.URL
	}

	service, err := NewBoostService("localhost:12345", relayEntries, testLog, relayTimeout)
	require.NoError(t, err)

	resp.boost = service
	return &resp
}

func (be *testBackend) post(t *testing.T, path string, payload any) *httptest.ResponseRecorder {
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", path, bytes.NewReader(payloadBytes))
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	be.boost.getRouter().ServeHTTP(rr, req)
	return rr
}

type mockRelay struct {
	Server       *httptest.Server
	RequestCount map[string]int
	mu           sync.Mutex

	HandlerOverride func(w http.ResponseWriter, req *http.Request) // used to make the relay do custom responses
}

func newMockRelay() *mockRelay {
	r := &mockRelay{
		RequestCount: make(map[string]int),
	}
	r.Server = httptest.NewServer(r.getRouter())
	return r
}

func (m *mockRelay) getRouter() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc(pathRegisterValidator, m.handleRegisterValidator)
	return m.requestCounterMiddleware(r)
}

func (m *mockRelay) requestCounterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			m.mu.Lock()
			url := r.URL.EscapedPath()
			m.RequestCount[url]++
			fmt.Println(url, m.RequestCount[url])
			m.mu.Unlock()

			next.ServeHTTP(w, r)
		},
	)
}

func (m *mockRelay) handleRegisterValidator(w http.ResponseWriter, req *http.Request) {
	if m.HandlerOverride != nil {
		m.HandlerOverride(w, req)
		return
	}

	payload := new(types.RegisterValidatorRequest)
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
