package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/mev-boost/types"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func _hexToBytes(hex string) []byte {
	bytes, err := hexutil.Decode(hex)
	if err != nil {
		panic(err)
	}
	return bytes
}

type testBackend struct {
	relays         []*mockRelay
	boostWebserver *BoostService
	boostRouter    http.Handler
}

func newTestBackend(t *testing.T, numRelays int, relayTimeouts RelayTimeouts) *testBackend {
	relays := make([]*mockRelay, numRelays)
	for i := 0; i < numRelays; i++ {
		relays[i] = newMockRelay()
	}
	return newTestBackendWithRelays(t, relays, relayTimeouts)
}

func newTestBackendWithRelays(t *testing.T, relays []*mockRelay, relayTimeouts RelayTimeouts) *testBackend {
	log := logrus.WithField("testing", true)
	resp := testBackend{
		relays: relays,
	}

	relayURLs := []string{}
	for i := 0; i < len(relays); i++ {
		relayURLs = append(relayURLs, resp.relays[i].Server.URL)
	}

	webserver, err := NewBoostService(":12345", relayURLs, log, relayTimeouts)
	require.NoError(t, err)
	boostRouter := webserver.getRouter()

	resp.boostWebserver = webserver
	resp.boostRouter = boostRouter
	return &resp
}

func (be *testBackend) post(t *testing.T, path string, payload any) *httptest.ResponseRecorder {
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", path, bytes.NewReader(payloadBytes))
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	be.boostRouter.ServeHTTP(rr, req)
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
	r.HandleFunc("/registerValidator", m.handleRegisterValidator)
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
