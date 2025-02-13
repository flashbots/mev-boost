// register_validator_test.go
package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	builderApiV1 "github.com/attestantio/go-builder-client/api/v1"
	"github.com/flashbots/mev-boost/server/mock"
	"github.com/flashbots/mev-boost/server/params"
	"github.com/flashbots/mev-boost/server/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// TestHandleRegisterValidator_EmptyList verifies that a valid registration returns status ok
func TestHandleRegisterValidator_EmptyList(t *testing.T) {
	relay := mock.NewRelay(t)
	defer relay.Server.Close()

	m := &BoostService{
		relays:           []types.RelayEntry{relay.RelayEntry},
		httpClientRegVal: *http.DefaultClient,
		log:              logrus.NewEntry(logrus.New()),
	}

	reqBody := bytes.NewBufferString("[]")
	req := httptest.NewRequest(http.MethodPost, "https://example.com"+params.PathRegisterValidator, reqBody)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	m.handleRegisterValidator(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "expected status ok")

	count := relay.GetRequestCount(params.PathRegisterValidator)
	require.Equal(t, 1, count)
}

// TestHandleRegisterValidator_NotEmptyList verifies that a non-empty list returns status ok
func TestHandleRegisterValidator_NotEmptyList(t *testing.T) {
	relay := mock.NewRelay(t)
	defer relay.Server.Close()

	m := &BoostService{
		relays:           []types.RelayEntry{relay.RelayEntry},
		httpClientRegVal: *http.DefaultClient,
		log:              logrus.NewEntry(logrus.New()),
	}

	validatorRegistrations := []builderApiV1.SignedValidatorRegistration{
		{
			Message: &builderApiV1.ValidatorRegistration{
				Timestamp: time.Unix(1, 0),
			},
		},
		{
			Message: &builderApiV1.ValidatorRegistration{
				Timestamp: time.Unix(2, 0),
			},
		},
	}

	encodedValidatorRegistrations, err := json.Marshal(validatorRegistrations)
	require.NoError(t, err)

	reqBody := bytes.NewBuffer(encodedValidatorRegistrations)
	req := httptest.NewRequest(http.MethodPost, "https://example.com"+params.PathRegisterValidator, reqBody)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	m.handleRegisterValidator(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "expected status ok")

	count := relay.GetRequestCount(params.PathRegisterValidator)
	require.Equal(t, 1, count)
}

// TestHandleRegisterValidator_InvalidJSON verifies that an invalid registration returns bad gateway
func TestHandleRegisterValidator_InvalidJSON(t *testing.T) {
	relay := mock.NewRelay(t)
	defer relay.Server.Close()

	m := &BoostService{
		relays:           []types.RelayEntry{relay.RelayEntry},
		httpClientRegVal: *http.DefaultClient,
		log:              logrus.NewEntry(logrus.New()),
	}

	reqBody := bytes.NewBufferString("invalid json")
	req := httptest.NewRequest(http.MethodPost, "https://example.com"+params.PathRegisterValidator, reqBody)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	m.handleRegisterValidator(rr, req)

	require.Equal(t, http.StatusBadGateway, rr.Code)

	count := relay.GetRequestCount(params.PathRegisterValidator)
	require.Equal(t, 1, count)
}

// TestHandleRegisterValidator_ValidSSZ verifies that a valid registration returns status ok
func TestHandleRegisterValidator_ValidSSZ(t *testing.T) {
	relay := mock.NewRelay(t)
	defer relay.Server.Close()

	m := &BoostService{
		relays:           []types.RelayEntry{relay.RelayEntry},
		httpClientRegVal: *http.DefaultClient,
		log:              logrus.NewEntry(logrus.New()),
	}

	validatorRegistrations := []builderApiV1.SignedValidatorRegistration{
		{
			Message: &builderApiV1.ValidatorRegistration{
				Timestamp: time.Unix(1, 0),
			},
		},
		{
			Message: &builderApiV1.ValidatorRegistration{
				Timestamp: time.Unix(2, 0),
			},
		},
	}

	// TODO(jtraglia): Use SSZ here when a SignedValidatorRegistrationList type exists.
	// See: https://github.com/attestantio/go-builder-client/pull/38
	encodedValidatorRegistrations, err := json.Marshal(validatorRegistrations)
	require.NoError(t, err)

	reqBody := bytes.NewBuffer(encodedValidatorRegistrations)
	req := httptest.NewRequest(http.MethodPost, "https://example.com"+params.PathRegisterValidator, reqBody)
	req.Header.Set("Content-Type", "application/octet-stream")

	rr := httptest.NewRecorder()
	m.handleRegisterValidator(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	count := relay.GetRequestCount(params.PathRegisterValidator)
	require.Equal(t, 1, count)
}

// TestHandleRegisterValidator_InvalidSSZ verifies that an invalid registration returns bad gateway
func TestHandleRegisterValidator_InvalidSSZ(t *testing.T) {
	relay := mock.NewRelay(t)
	defer relay.Server.Close()

	m := &BoostService{
		relays:           []types.RelayEntry{relay.RelayEntry},
		httpClientRegVal: *http.DefaultClient,
		log:              logrus.NewEntry(logrus.New()),
	}

	reqBody := bytes.NewBufferString("invalid ssz")
	req := httptest.NewRequest(http.MethodPost, "https://example.com"+params.PathRegisterValidator, reqBody)
	req.Header.Set("Content-Type", "application/octet-stream")

	rr := httptest.NewRecorder()
	m.handleRegisterValidator(rr, req)

	// TODO(jtraglia): Enable this when a SignedValidatorRegistrationList type exists.
	// See: https://github.com/attestantio/go-builder-client/pull/38
	// require.Equal(t, http.StatusBadGateway, rr.Code)

	count := relay.GetRequestCount(params.PathRegisterValidator)
	require.Equal(t, 1, count)
}

// TestHandleRegisterValidator_MultipleRelaysOneSuccess verifies that if one relay succeeds the response is ok
func TestHandleRegisterValidator_MultipleRelaysOneSuccess(t *testing.T) {
	badRelay := mock.NewRelay(t)
	defer badRelay.Server.Close()
	badRelay.OverrideHandleRegisterValidator(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "simulated failure", http.StatusInternalServerError)
	})

	relaySuccess := mock.NewRelay(t)
	defer relaySuccess.Server.Close()

	m := &BoostService{
		relays:           []types.RelayEntry{badRelay.RelayEntry, relaySuccess.RelayEntry},
		httpClientRegVal: *http.DefaultClient,
		log:              logrus.NewEntry(logrus.New()),
	}

	reqBody := bytes.NewBufferString("[]")
	req := httptest.NewRequest(http.MethodPost, "https://example.com"+params.PathRegisterValidator, reqBody)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	m.handleRegisterValidator(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	countBadRelay := badRelay.GetRequestCount(params.PathRegisterValidator)
	require.Equal(t, 1, countBadRelay)
	countSuccess := relaySuccess.GetRequestCount(params.PathRegisterValidator)
	require.Equal(t, 1, countSuccess)
}

// TestHandleRegisterValidator_AllFail verifies that if all relays fail the response is bad gateway
func TestHandleRegisterValidator_AllFail(t *testing.T) {
	badRelay1 := mock.NewRelay(t)
	defer badRelay1.Server.Close()
	badRelay1.OverrideHandleRegisterValidator(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "simulated failure 1", http.StatusInternalServerError)
	})

	badRelay2 := mock.NewRelay(t)
	defer badRelay2.Server.Close()
	badRelay2.OverrideHandleRegisterValidator(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "simulated failure 2", http.StatusInternalServerError)
	})

	m := &BoostService{
		relays:           []types.RelayEntry{badRelay1.RelayEntry, badRelay2.RelayEntry},
		httpClientRegVal: *http.DefaultClient,
		log:              logrus.NewEntry(logrus.New()),
	}

	reqBody := bytes.NewBufferString("[]")
	req := httptest.NewRequest(http.MethodPost, "https://example.com"+params.PathRegisterValidator, reqBody)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	m.handleRegisterValidator(rr, req)

	require.Equal(t, http.StatusBadGateway, rr.Code)

	countBadRelay1 := badRelay1.GetRequestCount(params.PathRegisterValidator)
	require.Equal(t, 1, countBadRelay1)
	countBadRelay2 := badRelay2.GetRequestCount(params.PathRegisterValidator)
	require.Equal(t, 1, countBadRelay2)
}

// TestHandleRegisterValidator_RelayNetworkError verifies that a network error results in bad gateway
func TestHandleRegisterValidator_RelayNetworkError(t *testing.T) {
	relay := mock.NewRelay(t)
	relay.Server.Close() // simulate network error

	m := &BoostService{
		relays:           []types.RelayEntry{relay.RelayEntry},
		httpClientRegVal: *http.DefaultClient,
		log:              logrus.NewEntry(logrus.New()),
	}

	reqBody := bytes.NewBufferString("[]")
	req := httptest.NewRequest(http.MethodPost, "https://example.com"+params.PathRegisterValidator, reqBody)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	m.handleRegisterValidator(rr, req)

	require.Equal(t, http.StatusBadGateway, rr.Code)
}

// TestHandleRegisterValidator_HeaderPropagation verifies that headers from the request are forwarded
func TestHandleRegisterValidator_HeaderPropagation(t *testing.T) {
	relay := mock.NewRelay(t)
	defer relay.Server.Close()

	headerChan := make(chan http.Header, 1)
	relay.OverrideHandleRegisterValidator(func(w http.ResponseWriter, req *http.Request) {
		headerChan <- req.Header
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})

	m := &BoostService{
		relays:           []types.RelayEntry{relay.RelayEntry},
		httpClientRegVal: *http.DefaultClient,
		log:              logrus.NewEntry(logrus.New()),
	}

	reqBody := bytes.NewBufferString("[]")
	req := httptest.NewRequest(http.MethodPost, "https://example.com"+params.PathRegisterValidator, reqBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Custom-Header", "custom-value")

	rr := httptest.NewRecorder()
	m.handleRegisterValidator(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	select {
	case capturedHeader := <-headerChan:
		require.Equal(t, "custom-value", capturedHeader.Get("X-Custom-Header"))
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for header capture")
	}
}
