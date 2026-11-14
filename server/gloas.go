package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"mime"
	"net/http"
	"strconv"
	"sync"
	"time"

	builderApiGloas "github.com/attestantio/go-builder-client/api/gloas"
	"github.com/attestantio/go-eth2-client/spec/gloas"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/flashbots/go-boost-utils/utils"
	"github.com/flashbots/mev-boost/config"
	"github.com/flashbots/mev-boost/server/params"
	"github.com/flashbots/mev-boost/server/types"
	gloasAPI "github.com/flashbots/mev-boost/server/types/gloas"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Gloas (ePBS) builder API. Each request is forwarded unchanged to the one builder whose
// auth data matches the request, and the builder's response is passed back.

// bidDeadlineMargin is taken off the deadline handed to the builder so its
// response still reaches the beacon node in time.
const bidDeadlineMargin = 50 * time.Millisecond

// maxGloasBodySize caps request bodies at 10 MiB, so an oversized request can't
// exhaust memory.
const maxGloasBodySize = 10_485_760 // 10 MiB = 10 * 1024 * 1024 bytes

var (
	errGloasConsensusVersion = errors.New("Eth-Consensus-Version header must be gloas")
	errMissingTimingHeaders  = errors.New("missing or invalid Date-Milliseconds or X-Timeout-Ms header")
	errMissingBody           = errors.New("missing request body")
	errBodyTooLarge          = fmt.Errorf("request body exceeds %d bytes", maxGloasBodySize)
	errAuthDataMismatch      = errors.New("auth data does not match any configured relay")
	errAuthSlotMismatch      = errors.New("auth slot does not match the request slot")
	errAuthSlotPassed        = errors.New("auth slot has already passed")
	errAuthSignature         = errors.New("request auth signature verification failed")
)

type gloasBody interface {
	UnmarshalSSZ(data []byte) error
	UnmarshalJSON(data []byte) error
}

type relayResponse struct {
	status           int
	body             []byte
	contentType      string
	consensusVersion string
}

func (m *BoostService) handleGetExecutionPayloadBid(w http.ResponseWriter, req *http.Request) {
	var (
		vars           = mux.Vars(req)
		slotStr        = vars["slot"]
		parentHashHex  = vars["parent_hash"]
		parentRootHex  = vars["parent_root"]
		proposerPubkey = vars["proposer_pubkey"]
	)
	log := m.log.WithFields(logrus.Fields{
		"method":         "getExecutionPayloadBid",
		"slot":           slotStr,
		"parentHash":     parentHashHex,
		"parentRoot":     parentRootHex,
		"proposerPubkey": proposerPubkey,
		"ua":             req.Header.Get(HeaderUserAgent),
	})
	log.Debug("handling request")

	slotValue, err := strconv.ParseUint(slotStr, 10, 64)
	if err != nil {
		m.respondGloasError(w, log, params.PathGetExecutionPayloadBid, http.StatusBadRequest, errInvalidSlot)
		return
	}
	slot := phase0.Slot(slotValue)
	// checks that both values are exactly 32 bytes long.
	if len(parentHashHex) != 66 || len(parentRootHex) != 66 {
		m.respondGloasError(w, log, params.PathGetExecutionPayloadBid, http.StatusBadRequest, errInvalidHash)
		return
	}
	deadline, err := parseRequestDeadline(req.Header)
	if err != nil {
		m.respondGloasError(w, log, params.PathGetExecutionPayloadBid, http.StatusBadRequest, err)
		return
	}

	auth := new(builderApiGloas.SignedBuilderRequestAuth)
	body, contentType, code, err := decodeGloasRequest(w, req, auth)
	if err != nil {
		m.respondGloasError(w, log, params.PathGetExecutionPayloadBid, code, err)
		return
	}
	if err := validateAuth(auth); err != nil {
		m.respondGloasError(w, log, params.PathGetExecutionPayloadBid, http.StatusBadRequest, err)
		return
	}
	if auth.Message.Slot != slot {
		m.respondGloasError(w, log, params.PathGetExecutionPayloadBid, http.StatusBadRequest, errAuthSlotMismatch)
		return
	}

	relay, code, err := m.authorizeGloasRequest(proposerPubkey, auth)
	if err != nil {
		m.respondGloasError(w, log, params.PathGetExecutionPayloadBid, code, err)
		return
	}
	log = log.WithField("relay", relay.RelayEntry.String())

	// a bid arriving later cannot make it into a block published before
	// the attestation deadline.
	m.relayConfigsLock.RLock()
	_, _, lateInSlotTimeMs := m.GetConfigForValidator(proposerPubkey)
	m.relayConfigsLock.RUnlock()
	slotStart := time.Unix(int64(m.genesisTime+uint64(slot)*config.SlotTimeSec), 0)
	deadline = time.UnixMilli(min(deadline.UnixMilli(), slotStart.UnixMilli()+int64(lateInSlotTimeMs))).Add(-bidDeadlineMargin)
	log.WithField("timeoutMs", time.Until(deadline).Milliseconds()).Debug("bid deadline")

	if time.Until(deadline) <= 0 {
		log.Warn("proposer deadline already passed, not requesting a bid")
		IncrementBeaconNodeStatus(strconv.Itoa(http.StatusNoContent), params.PathGetExecutionPayloadBid)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := fmt.Sprintf("/eth/v1/builder/execution_payload_bid/%d/%s/%s/%s", slot, parentHashHex, parentRootHex, proposerPubkey)
	header := gloasForwardHeader(req, contentType)
	ctx, cancel := context.WithDeadline(req.Context(), deadline)
	defer cancel()
	send := func(ctx context.Context) *relayResponse {
		return m.forwardGloasRequest(ctx, log, &m.httpClientGetHeader, relay.RelayEntry, params.PathGetExecutionPayloadBid, path, body, header)
	}

	var resp *relayResponse
	if relay.EnableTimingGames {
		resp = pollUntilDeadline(ctx, relay, slotStart, send)
	} else {
		resp = send(ctx)
	}
	if resp == nil {
		log.Warn("no response from relay, not returning a bid")
		IncrementBeaconNodeStatus(strconv.Itoa(http.StatusNoContent), params.PathGetExecutionPayloadBid)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if resp.status == http.StatusOK {
		logExecutionPayloadBid(log, relay.RelayEntry, resp)
	} else {
		log.WithField("statusCode", resp.status).Info("no bid from relay")
	}
	respondRelayResponse(w, log, params.PathGetExecutionPayloadBid, resp)
}

func (m *BoostService) handleSubmitBuilderPreferences(w http.ResponseWriter, req *http.Request) {
	proposerPubkey := mux.Vars(req)["proposer_pubkey"]
	log := m.log.WithFields(logrus.Fields{
		"method":         "submitBuilderPreferences",
		"proposerPubkey": proposerPubkey,
		"ua":             req.Header.Get(HeaderUserAgent),
	})
	log.Debug("handling request")

	prefs := new(builderApiGloas.BuilderPreferencesRequest)
	body, contentType, code, err := decodeGloasRequest(w, req, prefs)
	if err != nil {
		m.respondGloasError(w, log, params.PathSubmitBuilderPreferences, code, err)
		return
	}
	if err := validatePreferences(prefs); err != nil {
		m.respondGloasError(w, log, params.PathSubmitBuilderPreferences, http.StatusBadRequest, err)
		return
	}
	var currentSlot phase0.Slot
	if now := uint64(time.Now().Unix()); now > m.genesisTime {
		currentSlot = phase0.Slot((now - m.genesisTime) / config.SlotTimeSec)
	}
	// prefs for a slot that has already passed are stale
	if prefs.Auth.Message.Slot < currentSlot {
		m.respondGloasError(w, log, params.PathSubmitBuilderPreferences, http.StatusBadRequest, errAuthSlotPassed)
		return
	}
	relay, code, err := m.authorizeGloasRequest(proposerPubkey, prefs.Auth)
	if err != nil {
		m.respondGloasError(w, log, params.PathSubmitBuilderPreferences, code, err)
		return
	}
	log = log.WithFields(logrus.Fields{
		"relay":               relay.RelayEntry.String(),
		"slot":                prefs.Auth.Message.Slot,
		"maxExecutionPayment": prefs.Preferences.MaxExecutionPayment,
	})

	path := fmt.Sprintf("/eth/v1/builder/builder_preferences/%s", proposerPubkey)
	ctx, cancel := context.WithTimeout(req.Context(), m.httpClientRegVal.Timeout)
	defer cancel()
	resp := m.forwardGloasRequest(ctx, log, &m.httpClientRegVal, relay.RelayEntry, params.PathSubmitBuilderPreferences, path, body, gloasForwardHeader(req, contentType))
	if resp == nil {
		m.respondGloasError(w, log, params.PathSubmitBuilderPreferences, http.StatusInternalServerError, errNoSuccessfulRelayResponse)
		return
	}
	log.WithField("statusCode", resp.status).Info("forwarded builder preferences")
	respondRelayResponse(w, log, params.PathSubmitBuilderPreferences, resp)
}

func (m *BoostService) handleSubmitSignedBeaconBlock(w http.ResponseWriter, req *http.Request) {
	log := m.log.WithFields(logrus.Fields{
		"method": "submitSignedBeaconBlock",
		"ua":     req.Header.Get(HeaderUserAgent),
	})
	log.Debug("handling request")

	body, contentType, code, err := readGloasRequest(w, req)
	if err != nil {
		m.respondGloasError(w, log, params.PathSubmitSignedBeaconBlock, code, err)
		return
	}

	m.relayConfigsLock.RLock()
	relays := relayEntries(m.AllRelayConfigs())
	m.relayConfigsLock.RUnlock()

	header := gloasForwardHeader(req, contentType)
	acceptedCh := make(chan bool, len(relays))

	// we are forwarding the request to all the relays here just to avoid to store
	// the relays whos bid was requested for and then somehow getting data lost in
	// cache maybe due to restart. Only the relay whose bid won accepts it.
	// forwards must outlive the handler, which returns on the first acceptance
	ctx := context.WithoutCancel(req.Context())
	for _, relay := range relays {
		go func(relay types.RelayEntry) {
			ctx, cancel := context.WithTimeout(ctx, m.httpClientGetPayload.Timeout)
			defer cancel()
			resp := m.forwardGloasRequest(ctx, log, &m.httpClientGetPayload, relay, params.PathSubmitSignedBeaconBlock, params.PathSubmitSignedBeaconBlock, body, header)
			accepted := resp != nil && resp.status == http.StatusAccepted
			if resp != nil && !accepted {
				log.WithFields(logrus.Fields{"relay": relay.String(), "statusCode": resp.status}).Debug("relay did not accept the block")
			}
			acceptedCh <- accepted
		}(relay)
	}

	for range relays {
		if <-acceptedCh {
			log.Info("signed beacon block accepted by relay")
			IncrementBeaconNodeStatus(strconv.Itoa(http.StatusAccepted), params.PathSubmitSignedBeaconBlock)
			w.WriteHeader(http.StatusAccepted)
			return
		}
	}
	m.respondGloasError(w, log, params.PathSubmitSignedBeaconBlock, http.StatusInternalServerError, errNoSuccessfulRelayResponse)
}

// readGloasRequest checks the consensus version and content type headers and reads the body.
func readGloasRequest(w http.ResponseWriter, req *http.Request) (body []byte, contentType string, code int, err error) {
	if req.Header.Get(HeaderEthConsensusVersion) != EthConsensusVersionGloas {
		return nil, "", http.StatusBadRequest, errGloasConsensusVersion
	}
	contentType, err = requestContentType(req.Header)
	if err != nil {
		return nil, "", http.StatusUnsupportedMediaType, err
	}
	body, err = io.ReadAll(http.MaxBytesReader(w, req.Body, maxGloasBodySize))
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return nil, "", http.StatusBadRequest, errBodyTooLarge
		}
		return nil, "", http.StatusBadRequest, err
	}
	if len(body) == 0 {
		return nil, "", http.StatusBadRequest, errMissingBody
	}
	return body, contentType, 0, nil
}

// decodeGloasRequest reads the request and decodes the body into dst.
func decodeGloasRequest(w http.ResponseWriter, req *http.Request, dst gloasBody) (body []byte, contentType string, code int, err error) {
	body, contentType, code, err = readGloasRequest(w, req)
	if err != nil {
		return nil, "", code, err
	}
	if contentType == MediaTypeOctetStream {
		err = dst.UnmarshalSSZ(body)
	} else {
		err = dst.UnmarshalJSON(body)
	}
	if err != nil {
		return nil, "", http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err)
	}
	return body, contentType, 0, nil
}

// validateAuth checks the container limits.
func validateAuth(auth *builderApiGloas.SignedBuilderRequestAuth) error {
	if auth == nil || auth.Message == nil {
		return gloasAPI.ErrNilMessage
	}
	if len(auth.Message.Data) == 0 {
		return gloasAPI.ErrEmptyAuthData
	}
	if len(auth.Message.Data) > gloasAPI.MaxBuilderAuthDataSize {
		return gloasAPI.ErrAuthDataTooLarge
	}
	return nil
}

func validatePreferences(req *builderApiGloas.BuilderPreferencesRequest) error {
	if req == nil || req.Preferences == nil {
		return gloasAPI.ErrNilMessage
	}
	return validateAuth(req.Auth)
}

// authorizeGloasRequest finds the relay the request is addressed to and verifies the proposers signature.
func (m *BoostService) authorizeGloasRequest(proposerPubkey string, auth *builderApiGloas.SignedBuilderRequestAuth) (types.RelayConfig, int, error) {
	pubkey, err := utils.HexToPubkey(proposerPubkey)
	if err != nil {
		return types.RelayConfig{}, http.StatusBadRequest, errInvalidPubkey
	}

	m.relayConfigsLock.RLock()
	relayConfigs, _, _ := m.GetConfigForValidator(proposerPubkey)
	m.relayConfigsLock.RUnlock()

	var relay *types.RelayConfig
	for _, cfg := range relayConfigs {
		if cfg.RelayEntry.AuthData == string(auth.Message.Data) {
			relay = &cfg
			break
		}
	}
	if relay == nil {
		return types.RelayConfig{}, http.StatusBadRequest, errAuthDataMismatch
	}

	ok, err := gloasAPI.Verify(auth, pubkey, m.requestAuthDomain)
	if err != nil {
		return types.RelayConfig{}, http.StatusUnauthorized, err
	}
	if !ok {
		return types.RelayConfig{}, http.StatusUnauthorized, errAuthSignature
	}
	return *relay, 0, nil
}

// forwardGloasRequest posts the body to the relay and returns its response
func (m *BoostService) forwardGloasRequest(ctx context.Context, log *logrus.Entry, client *http.Client, relay types.RelayEntry, endpoint, path string, body []byte, header http.Header) *relayResponse {
	timeout := client.Timeout
	if timeout > bidDeadlineMargin {
		timeout -= bidDeadlineMargin
	}
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if timeout <= 0 || remaining < timeout {
			timeout = remaining
		}
	}

	url := relay.GetURI(path)
	log = log.WithField("url", url)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.WithError(err).Warn("error creating request")
		return nil
	}
	maps.Copy(req.Header, header)
	req.Header.Set(HeaderDateMilliseconds, strconv.FormatInt(time.Now().UTC().UnixMilli(), 10))
	req.Header.Set(HeaderTimeoutMs, strconv.FormatInt(timeout.Milliseconds(), 10))

	start := time.Now()
	resp, err := client.Do(req)
	RecordRelayLatency(endpoint, relay.URL.Hostname(), float64(time.Since(start).Milliseconds()))
	if err != nil {
		log.WithError(err).Warn("error calling relay")
		return nil
	}
	defer resp.Body.Close()
	RecordRelayStatusCode(strconv.Itoa(resp.StatusCode), endpoint, relay.URL.Hostname())

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Warn("error reading relay response")
		return nil
	}
	if resp.StatusCode >= http.StatusInternalServerError {
		log.WithFields(logrus.Fields{"statusCode": resp.StatusCode, "body": string(respBody)}).Warn("relay error response")
		return nil
	}
	return &relayResponse{
		status:           resp.StatusCode,
		body:             respBody,
		contentType:      resp.Header.Get(HeaderContentType),
		consensusVersion: resp.Header.Get(HeaderEthConsensusVersion),
	}
}

// pollUntilDeadline implements timing games for a relay, it waits for the configured first request
// time, then requests repeatedly until the context deadline and returns the most recently sent successful response.
func pollUntilDeadline(ctx context.Context, relay types.RelayConfig, slotStart time.Time, send func(context.Context) *relayResponse) *relayResponse {
	deadline, _ := ctx.Deadline()
	if relay.TargetFirstRequestMs > 0 {
		target := slotStart.Add(time.Duration(relay.TargetFirstRequestMs) * time.Millisecond)
		// target still in future and before the deadline.
		// todo: test scenario where the target is only a few milliseconds before the deadline
		// cuz it will send the req to the relay/builder with a near zero budget.
		// maybe we can add a min budget floor later....
		if target.Before(deadline) && time.Now().Before(target) {
			select {
			case <-time.After(time.Until(target)):
			case <-ctx.Done():
				return nil
			}
		}
	}
	if relay.FrequencyGetHeaderMs == 0 {
		return send(ctx)
	}

	var (
		mu         sync.Mutex
		wg         sync.WaitGroup
		latest     *relayResponse
		latestSent time.Time
		fallback   *relayResponse
	)
	poll := func() {
		defer wg.Done()
		sent := time.Now()
		resp := send(ctx)
		if resp == nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if resp.status != http.StatusOK {
			fallback = resp
			return
		}
		if sent.After(latestSent) {
			latest, latestSent = resp, sent
		}
	}

	frequency := time.Duration(relay.FrequencyGetHeaderMs) * time.Millisecond
	wg.Add(1)
	go poll()
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if time.Until(deadline) > frequency {
				wg.Add(1)
				go poll()
			}
		case <-ctx.Done():
			wg.Wait()
			if latest != nil {
				return latest
			}
			return fallback
		}
	}
}

func logExecutionPayloadBid(log *logrus.Entry, relay types.RelayEntry, resp *relayResponse) {
	var bid *gloas.SignedExecutionPayloadBid
	var err error
	if contentType, _, _ := mime.ParseMediaType(resp.contentType); contentType == MediaTypeOctetStream {
		bid = new(gloas.SignedExecutionPayloadBid)
		err = bid.UnmarshalSSZ(resp.body)
	} else {
		var wrapped gloasAPI.ExecutionPayloadBidResponse
		err = json.Unmarshal(resp.body, &wrapped)
		bid = wrapped.Data
	}
	if err != nil || bid == nil || bid.Message == nil {
		log.WithError(err).Warn("could not decode bid from relay")
		return
	}

	totalGwei := float64(bid.Message.Value) + float64(bid.Message.ExecutionPayment)
	RecordBidValue(relay.URL.Hostname(), totalGwei/1e9)
	RecordRelayLastSlot(relay.URL.Hostname(), uint64(bid.Message.Slot))
	log.WithFields(logrus.Fields{
		"blockHash":            bid.Message.BlockHash.String(),
		"builderIndex":         bid.Message.BuilderIndex,
		"valueGwei":            bid.Message.Value,
		"executionPaymentGwei": bid.Message.ExecutionPayment,
	}).Info("bid received")
}

func respondRelayResponse(w http.ResponseWriter, log *logrus.Entry, endpoint string, resp *relayResponse) {
	IncrementBeaconNodeStatus(strconv.Itoa(resp.status), endpoint)
	if resp.contentType != "" {
		w.Header().Set(HeaderContentType, resp.contentType)
	}
	if resp.consensusVersion != "" {
		w.Header().Set(HeaderEthConsensusVersion, resp.consensusVersion)
	} else if resp.status == http.StatusOK {
		w.Header().Set(HeaderEthConsensusVersion, EthConsensusVersionGloas)
	}
	w.WriteHeader(resp.status)
	if len(resp.body) > 0 {
		if _, err := w.Write(resp.body); err != nil {
			log.WithError(err).Error("error writing response")
		}
	}
}

func (m *BoostService) respondGloasError(w http.ResponseWriter, log *logrus.Entry, endpoint string, code int, err error) {
	log.WithError(err).WithField("statusCode", code).Warn("rejecting request")
	IncrementBeaconNodeStatus(strconv.Itoa(code), endpoint)
	m.respondError(w, code, err.Error())
}

func gloasForwardHeader(req *http.Request, contentType string) http.Header {
	header := http.Header{}
	header.Set(HeaderContentType, contentType)
	if accept := req.Header.Get(HeaderAccept); accept != "" {
		header.Set(HeaderAccept, accept)
	}
	header.Set(HeaderEthConsensusVersion, EthConsensusVersionGloas)
	header.Set(HeaderUserAgent, wrapUserAgent(UserAgent(req.Header.Get(HeaderUserAgent))))
	return header
}

// parseRequestDeadline returns the proposers deadline, Date-Milliseconds + X-Timeout-Ms,
// but no later than X-Timeout-Ms from now in case the proposers clock runs ahead.
// X-Timeout-Ms is capped at one slot, a bid is never useful for longer than that.
func parseRequestDeadline(header http.Header) (time.Time, error) {
	date, err := strconv.ParseInt(header.Get(HeaderDateMilliseconds), 10, 64)
	if err != nil {
		return time.Time{}, errMissingTimingHeaders
	}
	timeoutMs, err := strconv.ParseInt(header.Get(HeaderTimeoutMs), 10, 64)
	if err != nil || timeoutMs <= 0 {
		return time.Time{}, errMissingTimingHeaders
	}
	timeoutMs = min(timeoutMs, int64(config.SlotTimeSec)*1000)
	timeout := time.Duration(timeoutMs) * time.Millisecond
	deadline := time.UnixMilli(date).Add(timeout)
	if latest := time.Now().Add(timeout); latest.Before(deadline) {
		deadline = latest
	}
	return deadline, nil
}

func requestContentType(header http.Header) (string, error) {
	raw := header.Get(HeaderContentType)
	if raw == "" {
		return MediaTypeJSON, nil
	}
	mediaType, _, err := mime.ParseMediaType(raw)
	if err != nil {
		return "", err
	}
	switch mediaType {
	case MediaTypeJSON, MediaTypeOctetStream:
		return mediaType, nil
	}
	return "", types.ErrInvalidContentType
}
