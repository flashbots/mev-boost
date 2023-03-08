package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/attestantio/go-builder-client/api"
	"github.com/attestantio/go-eth2-client/api/v1/capella"
	"github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/go-utils/httplogger"
	"github.com/flashbots/mev-boost/config"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

var (
	errNoRelays                  = errors.New("no relays")
	errInvalidSlot               = errors.New("invalid slot")
	errInvalidHash               = errors.New("invalid hash")
	errInvalidPubkey             = errors.New("invalid pubkey")
	errNoSuccessfulRelayResponse = errors.New("no successful relay response")
	errServerAlreadyRunning      = errors.New("server already running")
)

var (
	nilHash     = types.Hash{}
	nilResponse = struct{}{}
)

type httpErrorResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// AuctionTranscript is the bid and blinded block received from the relay send to the relay monitor
type AuctionTranscript struct {
	Bid        *SignedBuilderBid               `json:"bid"`
	Acceptance *types.SignedBlindedBeaconBlock `json:"acceptance"`
}

// BoostServiceOpts provides all available options for use with NewBoostService
type BoostServiceOpts struct {
	Log                   *logrus.Entry
	ListenAddr            string
	Relays                []RelayEntry
	RelayMonitors         []*url.URL
	GenesisForkVersionHex string
	RelayCheck            bool
	RelayMinBid           types.U256Str

	RequestTimeoutGetHeader  time.Duration
	RequestTimeoutGetPayload time.Duration
	RequestTimeoutRegVal     time.Duration
	RequestMaxRetries        int
}

// BoostService - the mev-boost service
type BoostService struct {
	listenAddr    string
	relays        []RelayEntry
	relayMonitors []*url.URL
	log           *logrus.Entry
	srv           *http.Server
	relayCheck    bool
	relayMinBid   types.U256Str

	builderSigningDomain types.Domain
	httpClientGetHeader  http.Client
	httpClientGetPayload http.Client
	httpClientRegVal     http.Client
	requestMaxRetries    int

	bidsLock sync.Mutex
	bids     map[bidRespKey]bidResp // keeping track of bids, to log the originating relay on withholding
}

// NewBoostService created a new BoostService
func NewBoostService(opts BoostServiceOpts) (*BoostService, error) {
	if len(opts.Relays) == 0 {
		return nil, errNoRelays
	}

	builderSigningDomain, err := ComputeDomain(types.DomainTypeAppBuilder, opts.GenesisForkVersionHex, types.Root{}.String())
	if err != nil {
		return nil, err
	}

	return &BoostService{
		listenAddr:    opts.ListenAddr,
		relays:        opts.Relays,
		relayMonitors: opts.RelayMonitors,
		log:           opts.Log,
		relayCheck:    opts.RelayCheck,
		relayMinBid:   opts.RelayMinBid,
		bids:          make(map[bidRespKey]bidResp),

		builderSigningDomain: builderSigningDomain,
		httpClientGetHeader: http.Client{
			Timeout:       opts.RequestTimeoutGetHeader,
			CheckRedirect: httpClientDisallowRedirects,
		},
		httpClientGetPayload: http.Client{
			Timeout:       opts.RequestTimeoutGetPayload,
			CheckRedirect: httpClientDisallowRedirects,
		},
		httpClientRegVal: http.Client{
			Timeout:       opts.RequestTimeoutRegVal,
			CheckRedirect: httpClientDisallowRedirects,
		},
		requestMaxRetries: opts.RequestMaxRetries,
	}, nil
}

func (m *BoostService) respondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	resp := httpErrorResp{code, message}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		m.log.WithField("response", resp).WithError(err).Error("Couldn't write error response")
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (m *BoostService) respondOK(w http.ResponseWriter, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		m.log.WithField("response", response).WithError(err).Error("Couldn't write OK response")
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (m *BoostService) getRouter() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/", m.handleRoot)

	r.HandleFunc(pathStatus, m.handleStatus).Methods(http.MethodGet)
	r.HandleFunc(pathRegisterValidator, m.handleRegisterValidator).Methods(http.MethodPost)
	r.HandleFunc(pathGetHeader, m.handleGetHeader).Methods(http.MethodGet)
	r.HandleFunc(pathGetPayload, m.handleGetPayload).Methods(http.MethodPost)

	r.Use(mux.CORSMethodMiddleware(r))
	loggedRouter := httplogger.LoggingMiddlewareLogrus(m.log, r)
	return loggedRouter
}

// StartHTTPServer starts the HTTP server for this boost service instance
func (m *BoostService) StartHTTPServer() error {
	if m.srv != nil {
		return errServerAlreadyRunning
	}

	go m.startBidCacheCleanupTask()

	m.srv = &http.Server{
		Addr:    m.listenAddr,
		Handler: m.getRouter(),

		ReadTimeout:       time.Duration(config.ServerReadTimeoutMs) * time.Millisecond,
		ReadHeaderTimeout: time.Duration(config.ServerReadHeaderTimeoutMs) * time.Millisecond,
		WriteTimeout:      time.Duration(config.ServerWriteTimeoutMs) * time.Millisecond,
		IdleTimeout:       time.Duration(config.ServerIdleTimeoutMs) * time.Millisecond,

		MaxHeaderBytes: config.ServerMaxHeaderBytes,
	}

	err := m.srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (m *BoostService) startBidCacheCleanupTask() {
	for {
		time.Sleep(1 * time.Minute)
		m.bidsLock.Lock()
		for k, bidResp := range m.bids {
			if time.Since(bidResp.t) > 3*time.Minute {
				delete(m.bids, k)
			}
		}
		m.bidsLock.Unlock()
	}
}

func (m *BoostService) sendValidatorRegistrationsToRelayMonitors(payload []types.SignedValidatorRegistration) {
	log := m.log.WithField("method", "sendValidatorRegistrationsToRelayMonitors").WithField("numRegistrations", len(payload))
	for _, relayMonitor := range m.relayMonitors {
		go func(relayMonitor *url.URL) {
			url := GetURI(relayMonitor, pathRegisterValidator)
			log = log.WithField("url", url)
			_, err := SendHTTPRequest(context.Background(), m.httpClientRegVal, http.MethodPost, url, "", payload, nil)
			if err != nil {
				log.WithError(err).Warn("error calling registerValidator on relay monitor")
				return
			}
			log.Debug("sent validator registrations to relay monitor")
		}(relayMonitor)
	}
}

func (m *BoostService) sendAuctionTranscriptToRelayMonitors(transcript *AuctionTranscript) {
	log := m.log.WithField("method", "sendAuctionTranscriptToRelayMonitors")
	for _, relayMonitor := range m.relayMonitors {
		go func(relayMonitor *url.URL) {
			url := GetURI(relayMonitor, pathAuctionTranscript)
			log := log.WithField("url", url)
			_, err := SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodPost, url, UserAgent(""), transcript, nil)
			if err != nil {
				log.WithError(err).Warn("error sending auction transcript to relay monitor")
				return
			}
			log.Debug("sent auction transcript to relay monitor")
		}(relayMonitor)
	}
}

func (m *BoostService) handleRoot(w http.ResponseWriter, req *http.Request) {
	m.respondOK(w, nilResponse)
}

// handleStatus sends calls to the status endpoint of every relay.
// It returns OK if at least one returned OK, and returns error otherwise.
func (m *BoostService) handleStatus(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("X-MEVBoost-Version", config.Version)
	w.Header().Set("X-MEVBoost-ForkVersion", config.ForkVersion)
	if !m.relayCheck || m.CheckRelays() > 0 {
		m.respondOK(w, nilResponse)
	} else {
		m.respondError(w, http.StatusServiceUnavailable, "all relays are unavailable")
	}
}

// handleRegisterValidator - returns 200 if at least one relay returns 200, else 502
func (m *BoostService) handleRegisterValidator(w http.ResponseWriter, req *http.Request) {
	log := m.log.WithField("method", "registerValidator")
	log.Debug("registerValidator")

	payload := []types.SignedValidatorRegistration{}
	if err := DecodeJSON(req.Body, &payload); err != nil {
		m.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	ua := UserAgent(req.Header.Get("User-Agent"))
	log = log.WithFields(logrus.Fields{
		"numRegistrations": len(payload),
		"ua":               ua,
	})

	relayRespCh := make(chan error, len(m.relays))

	for _, relay := range m.relays {
		go func(relay RelayEntry) {
			url := relay.GetURI(pathRegisterValidator)
			log := log.WithField("url", url)

			_, err := SendHTTPRequest(context.Background(), m.httpClientRegVal, http.MethodPost, url, ua, payload, nil)
			relayRespCh <- err
			if err != nil {
				log.WithError(err).Warn("error calling registerValidator on relay")
				return
			}
		}(relay)
	}

	go m.sendValidatorRegistrationsToRelayMonitors(payload)

	for i := 0; i < len(m.relays); i++ {
		respErr := <-relayRespCh
		if respErr == nil {
			m.respondOK(w, nilResponse)
			return
		}
	}

	m.respondError(w, http.StatusBadGateway, errNoSuccessfulRelayResponse.Error())
}

// handleGetHeader requests bids from the relays
func (m *BoostService) handleGetHeader(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	slot := vars["slot"]
	parentHashHex := vars["parent_hash"]
	pubkey := vars["pubkey"]
	log := m.log.WithFields(logrus.Fields{
		"method":     "getHeader",
		"slot":       slot,
		"parentHash": parentHashHex,
		"pubkey":     pubkey,
	})
	log.Debug("getHeader")

	_slot, err := strconv.ParseUint(slot, 10, 64)
	if err != nil {
		m.respondError(w, http.StatusBadRequest, errInvalidSlot.Error())
		return
	}

	if len(pubkey) != 98 {
		m.respondError(w, http.StatusBadRequest, errInvalidPubkey.Error())
		return
	}

	if len(parentHashHex) != 66 {
		m.respondError(w, http.StatusBadRequest, errInvalidHash.Error())
		return
	}

	result := bidResp{}                           // the final response, containing the highest bid (if any)
	relays := make(map[BlockHashHex][]RelayEntry) // relays that sent the bid for a specific blockHash

	// Call the relays
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, relay := range m.relays {
		wg.Add(1)
		go func(relay RelayEntry) {
			defer wg.Done()
			path := fmt.Sprintf("/eth/v1/builder/header/%s/%s/%s", slot, parentHashHex, pubkey)
			url := relay.GetURI(path)
			log := log.WithField("url", url)
			responsePayload := new(GetHeaderResponse)
			code, err := SendHTTPRequest(context.Background(), m.httpClientGetHeader, http.MethodGet, url, UserAgent(req.Header.Get("User-Agent")), nil, responsePayload)
			if err != nil {
				log.WithError(err).Warn("error making request to relay")
				return
			}

			if code == http.StatusNoContent {
				log.Debug("no-content response")
				return
			}

			// Skip if invalid payload
			if responsePayload.IsInvalid() {
				return
			}

			blockHash := responsePayload.BlockHash()
			valueEth := weiBigIntToEthBigFloat(responsePayload.Value())
			log = log.WithFields(logrus.Fields{
				"blockNumber": responsePayload.BlockNumber(),
				"blockHash":   blockHash,
				"txRoot":      responsePayload.TransactionsRoot(),
				"value":       valueEth.Text('f', 18),
			})

			if relay.PublicKey.String() != responsePayload.Pubkey() {
				log.Errorf("bid pubkey mismatch. expected: %s - got: %s", relay.PublicKey.String(), responsePayload.Pubkey())
				return
			}

			// Verify the relay signature in the relay response
			ok, err := types.VerifySignature(responsePayload.Message(), m.builderSigningDomain, relay.PublicKey[:], responsePayload.Signature())
			if err != nil {
				log.WithError(err).Error("error verifying relay signature")
				return
			}
			if !ok {
				log.Error("failed to verify relay signature")
				return
			}

			// Verify response coherence with proposer's input data
			responseParentHash := responsePayload.ParentHash()
			if responseParentHash != parentHashHex {
				log.WithFields(logrus.Fields{
					"originalParentHash": parentHashHex,
					"responseParentHash": responseParentHash,
				}).Error("proposer and relay parent hashes are not the same")
				return
			}

			isZeroValue := responsePayload.Value().String() == "0"
			isEmptyListTxRoot := responsePayload.TransactionsRoot() == "0x7ffe241ea60187fdb0187bfa22de35d1f9bed7ab061d9401fd47e34a54fbede1"
			if isZeroValue || isEmptyListTxRoot {
				log.Warn("ignoring bid with 0 value")
				return
			}
			log.Debug("bid received")

			// Skip if value (fee) is lower than the minimum bid
			if responsePayload.Value().Cmp(m.relayMinBid.BigInt()) == -1 {
				log.Debug("ignoring bid below min-bid value")
				return
			}

			mu.Lock()
			defer mu.Unlock()

			// Remember which relays delivered which bids (multiple relays might deliver the top bid)
			relays[BlockHashHex(blockHash)] = append(relays[BlockHashHex(blockHash)], relay)

			// Compare the bid with already known top bid (if any)
			if !result.response.IsEmpty() {
				valueDiff := responsePayload.Value().Cmp(result.response.Value())
				if valueDiff == -1 { // current bid is less profitable than already known one
					return
				} else if valueDiff == 0 { // current bid is equally profitable as already known one. Use hash as tiebreaker
					previousBidBlockHash := result.response.BlockHash()
					if blockHash >= previousBidBlockHash {
						return
					}
				}
			}

			// Use this relay's response as mev-boost response because it's most profitable
			log.Debug("new best bid")
			result.response = *responsePayload
			result.blockHash = blockHash
			result.t = time.Now()
		}(relay)
	}

	// Wait for all requests to complete...
	wg.Wait()

	if result.blockHash == "" {
		log.Info("no bid received")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Log result
	valueEth := weiBigIntToEthBigFloat(result.response.Value())
	result.relays = relays[BlockHashHex(result.blockHash)]
	log.WithFields(logrus.Fields{
		"blockHash":   result.blockHash,
		"blockNumber": result.response.BlockNumber(),
		"txRoot":      result.response.TransactionsRoot(),
		"value":       valueEth.Text('f', 18),
		"relays":      strings.Join(RelayEntriesToStrings(result.relays), ", "),
	}).Info("best bid")

	// Remember the bid, for future logging in case of withholding
	bidKey := bidRespKey{slot: _slot, blockHash: result.blockHash}
	m.bidsLock.Lock()
	m.bids[bidKey] = result
	m.bidsLock.Unlock()

	// Return the bid
	m.respondOK(w, &result.response)
}

func (m *BoostService) processBellatrixPayload(w http.ResponseWriter, req *http.Request, log *logrus.Entry, payload *types.SignedBlindedBeaconBlock, body []byte) {
	if payload.Message == nil || payload.Message.Body == nil || payload.Message.Body.ExecutionPayloadHeader == nil {
		log.WithField("body", string(body)).Error("missing parts of the request payload from the beacon-node")
		m.respondError(w, http.StatusBadRequest, "missing parts of the payload")
		return
	}

	log = log.WithFields(logrus.Fields{
		"slot":       payload.Message.Slot,
		"blockHash":  payload.Message.Body.ExecutionPayloadHeader.BlockHash.String(),
		"parentHash": payload.Message.Body.ExecutionPayloadHeader.ParentHash.String(),
	})

	bidKey := bidRespKey{slot: payload.Message.Slot, blockHash: payload.Message.Body.ExecutionPayloadHeader.BlockHash.String()}
	m.bidsLock.Lock()
	originalBid := m.bids[bidKey]
	m.bidsLock.Unlock()
	if originalBid.blockHash == "" {
		log.Error("no bid for this getPayload payload found. was getHeader called before?")
	} else if len(originalBid.relays) == 0 {
		log.Warn("bid found but no associated relays")
	}

	// send bid and signed block to relay monitor
	go m.sendAuctionTranscriptToRelayMonitors(&AuctionTranscript{Bid: originalBid.response.BuilderBid(), Acceptance: payload})

	relays := originalBid.relays
	if len(relays) == 0 {
		log.Warn("originating relay not found, sending getPayload request to all relays")
		relays = m.relays
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	result := new(types.GetPayloadResponse)
	ua := UserAgent(req.Header.Get("User-Agent"))

	// Prepare the request context, which will be cancelled after the first successful response from a relay
	requestCtx, requestCtxCancel := context.WithCancel(context.Background())
	defer requestCtxCancel()

	for _, relay := range relays {
		wg.Add(1)
		go func(relay RelayEntry) {
			defer wg.Done()
			url := relay.GetURI(pathGetPayload)

			log := log.WithField("url", url)
			log.Debug("calling getPayload")

			responsePayload := new(types.GetPayloadResponse)
			_, err := SendHTTPRequestWithRetries(requestCtx, m.httpClientGetPayload, http.MethodPost, url, ua, payload, responsePayload, m.requestMaxRetries, log)
			if err != nil {
				if errors.Is(requestCtx.Err(), context.Canceled) {
					log.Info("request was cancelled") // this is expected, if payload has already been received by another relay
				} else {
					log.WithError(err).Error("error making request to relay")
				}
				return
			}

			if responsePayload.Data == nil || responsePayload.Data.BlockHash == nilHash {
				log.Error("response with empty data!")
				return
			}

			// Ensure the response blockhash matches the request
			if payload.Message.Body.ExecutionPayloadHeader.BlockHash != responsePayload.Data.BlockHash {
				log.WithFields(logrus.Fields{
					"responseBlockHash": responsePayload.Data.BlockHash.String(),
				}).Error("requestBlockHash does not equal responseBlockHash")
				return
			}

			// Ensure the response blockhash matches the response block
			calculatedBlockHash, err := types.CalculateHash(responsePayload.Data)
			if err != nil {
				log.WithError(err).Error("could not calculate block hash")
			} else if responsePayload.Data.BlockHash != calculatedBlockHash {
				log.WithFields(logrus.Fields{
					"calculatedBlockHash": calculatedBlockHash.String(),
					"responseBlockHash":   responsePayload.Data.BlockHash.String(),
				}).Error("responseBlockHash does not equal hash calculated from response block")
			}

			// Lock before accessing the shared payload
			mu.Lock()
			defer mu.Unlock()

			if requestCtx.Err() != nil { // request has been cancelled (or deadline exceeded)
				return
			}

			// Received successful response. Now cancel other requests and return immediately
			requestCtxCancel()
			*result = *responsePayload
			log.Info("received payload from relay")
		}(relay)
	}

	// Wait for all requests to complete...
	wg.Wait()

	// If no payload has been received from relay, log loudly about withholding!
	if result.Data == nil || result.Data.BlockHash == nilHash {
		originRelays := RelayEntriesToStrings(originalBid.relays)
		log.WithField("relays", strings.Join(originRelays, ", ")).Error("no payload received from relay!")
		m.respondError(w, http.StatusBadGateway, errNoSuccessfulRelayResponse.Error())
		return
	}

	m.respondOK(w, result)
}

func (m *BoostService) processCapellaPayload(w http.ResponseWriter, req *http.Request, log *logrus.Entry, payload *capella.SignedBlindedBeaconBlock, body []byte) {
	if payload.Message == nil || payload.Message.Body == nil || payload.Message.Body.ExecutionPayloadHeader == nil {
		log.WithField("body", string(body)).Error("missing parts of the request payload from the beacon-node")
		m.respondError(w, http.StatusBadRequest, "missing parts of the payload")
		return
	}

	log = log.WithFields(logrus.Fields{
		"slot":       payload.Message.Slot,
		"blockHash":  payload.Message.Body.ExecutionPayloadHeader.BlockHash.String(),
		"parentHash": payload.Message.Body.ExecutionPayloadHeader.ParentHash.String(),
	})

	bidKey := bidRespKey{slot: uint64(payload.Message.Slot), blockHash: payload.Message.Body.ExecutionPayloadHeader.BlockHash.String()}
	m.bidsLock.Lock()
	originalBid := m.bids[bidKey]
	m.bidsLock.Unlock()
	if originalBid.blockHash == "" {
		log.Error("no bid for this getPayload payload found. was getHeader called before?")
	} else if len(originalBid.relays) == 0 {
		log.Warn("bid found but no associated relays")
	}

	// send bid and signed block to relay monitor with capella payload
	// go m.sendAuctionTranscriptToRelayMonitors(&AuctionTranscript{Bid: originalBid.response.Data, Acceptance: payload})

	relays := originalBid.relays
	if len(relays) == 0 {
		log.Warn("originating relay not found, sending getPayload request to all relays")
		relays = m.relays
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	result := new(api.VersionedExecutionPayload)
	ua := UserAgent(req.Header.Get("User-Agent"))

	// Prepare the request context, which will be cancelled after the first successful response from a relay
	requestCtx, requestCtxCancel := context.WithCancel(context.Background())
	defer requestCtxCancel()

	for _, relay := range relays {
		wg.Add(1)
		go func(relay RelayEntry) {
			defer wg.Done()
			url := relay.GetURI(pathGetPayload)
			log := log.WithField("url", url)
			log.Debug("calling getPayload")

			responsePayload := new(api.VersionedExecutionPayload)
			_, err := SendHTTPRequestWithRetries(requestCtx, m.httpClientGetPayload, http.MethodPost, url, ua, payload, responsePayload, m.requestMaxRetries, log)
			if err != nil {
				if errors.Is(requestCtx.Err(), context.Canceled) {
					log.Info("request was cancelled") // this is expected, if payload has already been received by another relay
				} else {
					log.WithError(err).Error("error making request to relay")
				}
				return
			}

			if responsePayload.Capella == nil || types.Hash(responsePayload.Capella.BlockHash) == nilHash {
				log.Error("response with empty data!")
				return
			}

			// Ensure the response blockhash matches the request
			if payload.Message.Body.ExecutionPayloadHeader.BlockHash != responsePayload.Capella.BlockHash {
				log.WithFields(logrus.Fields{
					"responseBlockHash": responsePayload.Capella.BlockHash.String(),
				}).Error("requestBlockHash does not equal responseBlockHash")
				return
			}

			// Ensure the response blockhash matches the response block
			calculatedBlockHash, err := ComputeBlockHash(responsePayload.Capella)
			if err != nil {
				log.WithError(err).Error("could not calculate block hash")
			} else if responsePayload.Capella.BlockHash != calculatedBlockHash {
				log.WithFields(logrus.Fields{
					"calculatedBlockHash": calculatedBlockHash.String(),
					"responseBlockHash":   responsePayload.Capella.BlockHash.String(),
				}).Error("responseBlockHash does not equal hash calculated from response block")
			}

			// Lock before accessing the shared payload
			mu.Lock()
			defer mu.Unlock()

			if requestCtx.Err() != nil { // request has been cancelled (or deadline exceeded)
				return
			}

			// Received successful response. Now cancel other requests and return immediately
			requestCtxCancel()
			*result = *responsePayload
			log.Info("received payload from relay")
		}(relay)
	}

	// Wait for all requests to complete...
	wg.Wait()

	// If no payload has been received from relay, log loudly about withholding!
	if result.Capella == nil || types.Hash(result.Capella.BlockHash) == nilHash {
		originRelays := RelayEntriesToStrings(originalBid.relays)
		log.WithField("relays", strings.Join(originRelays, ", ")).Error("no payload received from relay!")
		m.respondError(w, http.StatusBadGateway, errNoSuccessfulRelayResponse.Error())
		return
	}

	m.respondOK(w, result)
}

func (m *BoostService) handleGetPayload(w http.ResponseWriter, req *http.Request) {
	log := m.log.WithField("method", "getPayload")
	log.Debug("getPayload")

	// Read the body first, so we can log it later on error
	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Error("could not read body of request from the beacon node")
		m.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Decode the body now
	payload := new(capella.SignedBlindedBeaconBlock)
	if err := DecodeJSON(bytes.NewReader(body), payload); err != nil {
		log.Debug("could not decode Capella request payload, attempting to decode body into Bellatrix payload")
		payload := new(types.SignedBlindedBeaconBlock)
		if err := DecodeJSON(bytes.NewReader(body), payload); err != nil {
			log.WithError(err).WithField("body", string(body)).Error("could not decode request payload from the beacon-node (signed blinded beacon block)")
			m.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		m.processBellatrixPayload(w, req, log, payload, body)
		return
	}
	m.processCapellaPayload(w, req, log, payload, body)
}

// CheckRelays sends a request to each one of the relays previously registered to get their status
func (m *BoostService) CheckRelays() int {
	var wg sync.WaitGroup
	var numSuccessRequestsToRelay uint32

	for _, r := range m.relays {
		wg.Add(1)

		go func(relay RelayEntry) {
			defer wg.Done()
			url := relay.GetURI(pathStatus)
			log := m.log.WithField("url", url)
			log.Debug("checking relay status")

			code, err := SendHTTPRequest(context.Background(), m.httpClientGetHeader, http.MethodGet, url, "", nil, nil)
			if err != nil {
				log.WithError(err).Error("relay status error - request failed")
				return
			}
			if code == http.StatusOK {
				log.Debug("relay status OK")
			} else {
				log.Errorf("relay status error - unexpected status code %d", code)
				return
			}

			// Success: increase counter and cancel all pending requests to other relays
			atomic.AddUint32(&numSuccessRequestsToRelay, 1)
		}(r)
	}

	// At the end, wait for every routine and return status according to relay's ones.
	wg.Wait()
	return int(numSuccessRequestsToRelay)
}
