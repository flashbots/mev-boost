package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/mev-boost/config/rcm"
	"github.com/flashbots/mev-boost/config/rcp"
	"github.com/flashbots/mev-boost/config/rcp/rcptest"
	"github.com/flashbots/mev-boost/config/relay"
	"github.com/flashbots/mev-boost/config/relay/reltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testBackendCfg struct {
	useRelaysWithRandomKeys bool
	disableLogs             bool
}

type testBackendOpt func(cfg *testBackendCfg)

func withRandomRelayKeys() testBackendOpt {
	return func(cfg *testBackendCfg) {
		cfg.useRelaysWithRandomKeys = true
	}
}

func withDiscardedOutput() testBackendOpt {
	return func(cfg *testBackendCfg) {
		cfg.disableLogs = true
	}
}

type testBackend struct {
	boost  *BoostService
	relays []*mockRelay
}

// newTestBackend creates a new backend, initializes mock relays, registers them and return the instance
func newTestBackend(tb testing.TB, numRelays int, relayTimeout time.Duration, opt ...testBackendOpt) *testBackend {
	tb.Helper()

	cfg := &testBackendCfg{}
	for _, o := range opt {
		o(cfg)
	}

	backend := testBackend{
		relays: make([]*mockRelay, numRelays),
	}

	relays := relay.NewRelaySet()
	for i := 0; i < numRelays; i++ {
		// Create a mock relay
		if cfg.useRelaysWithRandomKeys {
			backend.relays[i] = randomMockRelay(tb)
		} else {
			backend.relays[i] = staticMockRelay(tb)
		}

		relays.Add(backend.relays[i].RelayEntry)
	}

	relayConfigurator, err := rcm.New(rcm.NewRegistryCreator(rcp.NewDefault(relays).FetchConfig))
	require.NoError(tb, err)

	log := testLog
	if cfg.disableLogs {
		log.Logger.SetOutput(io.Discard)
	}

	opts := BoostServiceOpts{
		Log:                      log,
		ListenAddr:               "localhost:12345",
		RelayConfigurator:        relayConfigurator,
		GenesisForkVersionHex:    "0x00000000",
		RelayCheck:               true,
		RelayMinBid:              types.IntToU256(12345),
		RequestTimeoutGetHeader:  relayTimeout,
		RequestTimeoutGetPayload: relayTimeout,
		RequestTimeoutRegVal:     relayTimeout,
	}
	service, err := NewBoostService(opts)
	require.NoError(tb, err)

	backend.boost = service
	return &backend
}

func (be *testBackend) close() {
	for _, r := range be.relays {
		r.Server.Close()
	}
}

func (be *testBackend) request(tb testing.TB, method, path string, payload any) *httptest.ResponseRecorder {
	tb.Helper()
	var req *http.Request
	var err error

	if payload == nil {
		req, err = http.NewRequest(method, path, bytes.NewReader(nil))
	} else {
		payloadBytes, err2 := json.Marshal(payload)
		require.NoError(tb, err2)
		req, err = http.NewRequest(method, path, bytes.NewReader(payloadBytes))
	}

	require.NoError(tb, err)
	rr := httptest.NewRecorder()
	be.boost.getRouter().ServeHTTP(rr, req)
	//tb.Cleanup(func() {
	//	require.NoError(tb, rr.Result().Body.Close())
	//})

	return rr
}

func (be *testBackend) requestRegisterValidator(tb testing.TB, path string, payload any) *httptest.ResponseRecorder {
	tb.Helper()

	return be.request(tb, http.MethodPost, path, payload)
}

func (be *testBackend) requestGetHeader(tb testing.TB, path string) *httptest.ResponseRecorder {
	tb.Helper()

	return be.request(tb, http.MethodGet, path, nil)
}

func (be *testBackend) requestGetPayload(tb testing.TB, path string, payload any) *httptest.ResponseRecorder {
	tb.Helper()

	return be.request(tb, http.MethodPost, path, payload)
}

func (be *testBackend) stubRelayGetBellatrixHeaderResponse(tb testing.TB, index int, value uint64) {
	tb.Helper()

	r := be.relayByIndex(tb, index)
	r.GetHeaderResponse = r.MakeGetHeaderResponse(
		value,
		"0xc457e7cedd8bf0c16630dea852b20fb387fdb71c5ab369529ec09011371b22e1",
		"0xe8b9bd82aa0e957736c5a029903e53d581edf451e28ab274f4ba314c442e35a4",
		r.RelayEntry.PublicKey().String(),
		spec.DataVersionBellatrix,
	)
}

func (be *testBackend) stubRelayGetBellatrixPayloadResponse(
	tb testing.TB, index int, payload *types.SignedBlindedBeaconBlock,
) {
	tb.Helper()

	r := be.relayByIndex(tb, index)
	r.GetBellatrixPayloadResponse = &types.GetPayloadResponse{
		Data: blindedBlockToExecutionPayloadBellatrix(payload),
	}
}

func (be *testBackend) stubRelayHTTPServerWithTemporaryRedirect(tb testing.TB, index int) {
	tb.Helper()

	relayMock := be.relayByIndex(tb, index)
	relayMock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, relayMock.Server.URL, http.StatusTemporaryRedirect)
	}))
}

func (be *testBackend) stubRelayEntryWithARandomPublicKey(tb testing.TB, index int) relay.Entry {
	tb.Helper()

	return be.stubRelayEntryWithUser(tb, index, url.User(reltest.RandomBLSPublicKey(tb).String()))
}

func (be *testBackend) stubRelayEntry(tb testing.TB, index int) relay.Entry {
	tb.Helper()

	r := be.relayByIndex(tb, index)

	return be.stubRelayEntryWithUser(tb, index, r.RelayEntry.RelayURL().User)
}

func (be *testBackend) stubRelayEntryWithUser(tb testing.TB, index int, user *url.Userinfo) relay.Entry {
	tb.Helper()

	r := be.relayByIndex(tb, index)

	forgedURL, err := url.ParseRequestURI(r.Server.URL)
	require.NoError(tb, err)

	forgedURL.User = user

	relayEntry, err := relay.NewRelayEntry(forgedURL.String())
	require.NoError(tb, err)

	r.RelayEntry = relayEntry

	return relayEntry
}

func (be *testBackend) relayByIndex(tb testing.TB, index int) *mockRelay {
	tb.Helper()

	require.Truef(tb, index < len(be.relays), "relay index %d is out of range", index)

	return be.relays[index]
}

func (be *testBackend) stubConfigurator(tb testing.TB, proposerRelays map[string]relay.Set, defaultRelays relay.Set) {
	tb.Helper()

	opts := make([]rcptest.MockOption, 0, len(proposerRelays)+1)
	if defaultRelays != nil {
		opts = append(opts, rcptest.WithDefaultRelays(defaultRelays))
	}

	for pubKey, relays := range proposerRelays {
		opts = append(opts, rcptest.WithProposerRelays(pubKey, relays))
	}

	relayConfigProvider := rcptest.MockRelayConfigProvider(opts...)

	cm, err := rcm.New(rcm.NewRegistryCreator(relayConfigProvider))
	require.NoError(tb, err)

	be.boost.relayConfigurator = cm
}

func (be *testBackend) relaySet(tb testing.TB, indexes ...int) relay.Set {
	tb.Helper()

	list := make(relay.List, 0, len(indexes))
	for i := range indexes {
		list = append(list, be.relayByIndex(tb, i).RelayEntry)
	}

	return reltest.RelaySetFromList(list)
}

func assertRequestWasSuccessful(tb testing.TB, rr *httptest.ResponseRecorder) {
	tb.Helper()

	require.Equal(tb, http.StatusOK, rr.Code, rr.Body.String())
}

func assertRelayReturnedNoContent(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func assertBadGateway(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()

	assert.Equal(t, http.StatusBadGateway, rr.Code)
}

func assertRelaysReceivedRequest(t *testing.T, sut *testBackend) func(string, ...int) {
	t.Helper()

	return func(path string, index ...int) {
		for _, i := range index {
			r := sut.relayByIndex(t, i)
			assert.Equalf(t, 1, r.GetRequestCount(path),
				"path: %s: relay: %d: %s",
				path, i, r.RelayEntry.RelayURL())
		}
	}
}

func assertRelaysReceivedAtLeastOneRequest(t *testing.T, sut *testBackend) func(string, ...int) {
	t.Helper()

	return func(path string, index ...int) {
		var requestsCount int

		for _, i := range index {
			requestsCount += sut.relayByIndex(t, i).GetRequestCount(path)
		}

		assert.GreaterOrEqualf(t, requestsCount, 1, "at least one response from a relay expected")
	}
}
