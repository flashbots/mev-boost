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
	"github.com/flashbots/builder/types"
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

func _HexToHash(s string) (ret types.Hash) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		testLog.Error(err, " _HexToHash: ", s)
		panic(err)
	}
	return ret
}

func _HexToAddress(s string) (ret types.Address) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		testLog.Error(err, " _HexToAddress: ", s)
		panic(err)
	}
	return ret
}

func _HexToPubkey(s string) (ret types.PublicKey) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		testLog.Error(err, " _HexToPubkey: ", s)
		panic(err)
	}
	return
}

func _HexToSignature(s string) (ret types.Signature) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		testLog.Error(err, " _HexToSignature: ", s)
		panic(err)
	}
	return
}

type testBackend struct {
	boost  *BoostService
	relays []*mockRelay
}

func newTestBackend(t *testing.T, numRelays int, relayTimeout time.Duration) *testBackend {
	var err error
	resp := testBackend{
		relays: make([]*mockRelay, numRelays),
	}

	relayEntries := make([]*RelayEntry, numRelays)
	for i := 0; i < numRelays; i++ {
		resp.relays[i] = newMockRelay()
		relayEntries[i], err = NewRelayEntry(resp.relays[i].Server.URL)
		require.NoError(t, err)
	}

	service, err := NewBoostService("localhost:12345", relayEntries, testLog, relayTimeout)
	require.NoError(t, err)

	resp.boost = service
	return &resp
}

func (be *testBackend) request(t *testing.T, method string, path string, payload any) *httptest.ResponseRecorder {
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

type mockRelay struct {
	Server       *httptest.Server
	RequestCount map[string]int
	mu           sync.Mutex

	ResponseDelay      time.Duration
	HandlerOverride    func(w http.ResponseWriter, req *http.Request) // used to make the relay do custom responses
	GetHeaderResponse  *types.GetHeaderResponse                       // hard-coded response payload (used if no HandlerOverride exists)
	GetPayloadResponse *types.GetPayloadResponse                      // hard-coded response payload (used if no HandlerOverride exists)
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
	r.HandleFunc(pathStatus, m.handleStatus).Methods(http.MethodGet)
	r.HandleFunc(pathRegisterValidator, m.handleRegisterValidator).Methods(http.MethodPost)
	r.HandleFunc(pathGetHeader, m.handleGetHeader).Methods(http.MethodGet)
	r.HandleFunc(pathGetPayload, m.handleGetPayload).Methods(http.MethodPost)
	return m.testMiddleware(r)
}

func (m *mockRelay) getRequestCount(path string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.RequestCount[path]
}

func (m *mockRelay) testMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// Request counter
			m.mu.Lock()
			url := r.URL.EscapedPath()
			m.RequestCount[url]++
			// fmt.Println(url, m.RequestCount[url])
			m.mu.Unlock()

			// Artificial Delay
			if m.ResponseDelay > 0 {
				time.Sleep(m.ResponseDelay)
			}

			next.ServeHTTP(w, r)
		},
	)
}

func (m *mockRelay) handleStatus(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{}`)
}

func (m *mockRelay) handleRegisterValidator(w http.ResponseWriter, req *http.Request) {
	if m.HandlerOverride != nil {
		m.HandlerOverride(w, req)
		return
	}

	payload := new(types.SignedValidatorRegistration)
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func makeGetHeaderResponse(value uint64) *types.GetHeaderResponse {
	return &types.GetHeaderResponse{
		Version: "bellatrix",
		Data: &types.SignedBuilderBid{
			Message: &types.BuilderBid{
				Header: &types.ExecutionPayloadHeader{
					BlockHash: _HexToHash("0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7"),
				},
				Value:  types.IntToU256(value),
				Pubkey: _HexToPubkey("0xe08851236b8a3fbd0645cc21247258cd6cfd30e085bb5a9a54a9dd9373737f29252f0e42d97559e889d6c5b750fd8086"),
			},
			Signature: _HexToSignature("0x1ea6d65fb0305ef317d20ce830019f6d8bb0da231d65658786c1cf68429a0e8f1f8776a7fea90673f05e7016fe1c762763544e0231ef8b4e30b77eedc1b7f54da45889c9d5718c3912b5a689711455836257a7608665cb70869cb3a647ba0f7b"),
		},
	}
}

func (m *mockRelay) handleGetHeader(w http.ResponseWriter, req *http.Request) {
	if m.HandlerOverride != nil {
		m.HandlerOverride(w, req)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := makeGetHeaderResponse(12345)
	if m.GetHeaderResponse != nil {
		response = m.GetHeaderResponse
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func makeGetPayloadResponse() *types.GetPayloadResponse {
	return &types.GetPayloadResponse{
		Version: "bellatrix",
		Data: &types.ExecutionPayload{
			ParentHash:   _HexToHash("0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7"),
			BlockHash:    _HexToHash("0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab1"),
			BlockNumber:  12345,
			FeeRecipient: _HexToAddress("0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941"),
		},
	}
}

func (m *mockRelay) handleGetPayload(w http.ResponseWriter, req *http.Request) {
	if m.HandlerOverride != nil {
		m.HandlerOverride(w, req)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := makeGetPayloadResponse()
	if m.GetPayloadResponse != nil {
		response = m.GetPayloadResponse
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
