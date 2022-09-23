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

	"github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/go-utils/httplogger"
	"github.com/flashbots/mev-boost/config"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var (
	errInvalidSlot               = errors.New("invalid slot")
	errInvalidHash               = errors.New("invalid hash")
	errInvalidPubkey             = errors.New("invalid pubkey")
	errNoSuccessfulRelayResponse = errors.New("no successful relay response")

	errServerAlreadyRunning = errors.New("server already running")
)

var nilHash = types.Hash{}
var nilResponse = struct{}{}

type httpErrorResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// BoostServiceOpts provides all available options for use with NewBoostService
type BoostServiceOpts struct {
	Log                   *logrus.Entry
	ListenAddr            string
	Relays                []RelayEntry
	RelayMonitors         []*url.URL
	GenesisForkVersionHex string
	RelayCheck            bool

	RequestTimeoutGetHeader  time.Duration
	RequestTimeoutGetPayload time.Duration
	RequestTimeoutRegVal     time.Duration
	MetricOpts               MetricOpts
}

// BoostService - the mev-boost service
type BoostService struct {
	listenAddr    string
	relays        []RelayEntry
	relayMonitors []*url.URL
	log           *logrus.Entry
	srv           *http.Server
	relayCheck    bool

	builderSigningDomain types.Domain
	httpClientGetHeader  http.Client
	httpClientGetPayload http.Client
	httpClientRegVal     http.Client

	bidsLock sync.Mutex
	bids     map[bidRespKey]bidResp // keeping track of bids, to log the originating relay on withholding

	Metrics        Metrics
	metricRegistry *prometheus.Registry
}

// Metrics contains all metric registries
type Metrics struct {
	metricsMev          *MevMetrics
	metricsOutboundHTTP *OutboundHTTPMetrics
	metricsInboundHTTP  *InboundHTTPMetrics
}

// NewMetrics constructs a metrics registry configuration and initialises static metrics
func NewMetrics(metricsRegistry *prometheus.Registry, opts MetricOpts, relays []RelayEntry) Metrics {
	mevMetrics := NewMevMetrics(metricsRegistry)
	if GenesisForkVersionDec, err := strconv.ParseInt(opts.GenesisForkVersionHex, 16, 64); err == nil {
		mevMetrics.genesisForkVersion.Set(float64(GenesisForkVersionDec))
	}
	mevMetrics.version.With(prometheus.Labels{labelVersion: opts.ServiceVersion}).Set(1)
	mevMetrics.relayCount.Set(float64(len(relays)))
	for _, relay := range relays {
		mevMetrics.validatorIdentities.With(prometheus.Labels{labelPubkey: relay.PublicKey.String()}).Set(1)
		mevMetrics.relays.WithLabelValues(relay.URL.Hostname()).Set(1)
	}
	return Metrics{
		metricsMev:          mevMetrics,
		metricsOutboundHTTP: NewOutboundHTTPMetrics(metricsRegistry),
		metricsInboundHTTP:  NewInboundHTTPMetrics(metricsRegistry),
	}
}

// MetricOpts exposes configuration details for prometheus handler
type MetricOpts struct {
	Enabled               bool
	Registry              *prometheus.Registry
	ServiceVersion        string
	GenesisForkVersionHex string
}

// NewMetricOpts constructs a MetricOpts with sensible defaults
func NewMetricOpts(enabled bool, registry *prometheus.Registry) MetricOpts {
	if registry == nil {
		registry = prometheus.NewRegistry()
	}
	return MetricOpts{
		Enabled:  enabled,
		Registry: registry,
	}
}

// NewBoostService created a new BoostService
func NewBoostService(opts BoostServiceOpts) (*BoostService, error) {
	if len(opts.Relays) == 0 {
		return nil, errors.New("no relay_count")
	}

	builderSigningDomain, err := ComputeDomain(types.DomainTypeAppBuilder, opts.GenesisForkVersionHex, types.Root{}.String())
	if err != nil {
		return nil, err
	}

	var metricsRegistry = opts.MetricOpts.Registry
	if opts.MetricOpts.Registry == nil {
		metricsRegistry = prometheus.NewRegistry()
	}
	return &BoostService{
		listenAddr:           opts.ListenAddr,
		relays:               opts.Relays,
		relayMonitors:        opts.RelayMonitors,
		log:                  opts.Log.WithField("module", "service"),
		relayCheck:           opts.RelayCheck,
		bids:                 make(map[bidRespKey]bidResp),
		metricRegistry:       opts.MetricOpts.Registry,
		Metrics:              NewMetrics(metricsRegistry, opts.MetricOpts, opts.Relays),
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

	if m.metricRegistry != nil {
		fmt.Println("enabled")
		log := m.log.WithField("path", pathMetrics)
		log.Infof("prometheus metrics exposed at %s", pathMetrics)
		r.Handle(pathMetrics, promhttp.HandlerFor(m.metricRegistry, promhttp.HandlerOpts{
			ErrorLog: m.log,
		})).Methods(http.MethodGet)
	}

	return Chain(
		httplogger.LoggingMiddlewareLogrus(m.log, r),
		Middleware(mux.CORSMethodMiddleware(r)),
		InboundHTTPMetricMiddleware(m.Metrics.metricsInboundHTTP),
	)
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
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (m *BoostService) startBidCacheCleanupTask() {
	for {
		m.Metrics.metricsMev.bids.Set(float64(len(m.bids)))
		time.Sleep(time.Minute)
		m.bidsLock.Lock()
		for k, bidResp := range m.bids {
			if time.Since(bidResp.t) > 3*time.Minute {
				delete(m.bids, k)
				continue
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
			_, err := SendHTTPRequest(context.Background(), m.httpClientRegVal, http.MethodPost, url, UserAgent(""), payload, nil)
			if err != nil {
				log.WithError(err).Warn("error calling registerValidator on relay monitor")
				return
			}
			log.Debug("sent validator registrations to relay monitor")
		}(relayMonitor)
	}
}

func (m *BoostService) handleRoot(w http.ResponseWriter, req *http.Request) {
	m.respondOK(w, nilResponse)
}

// handleStatus sends calls to the status endpoint of every relay.
// It returns OK if at least one returned OK, and returns error otherwise.
func (m *BoostService) handleStatus(w http.ResponseWriter, req *http.Request) {
	if !m.relayCheck || m.CheckRelays() > 0 {
		m.respondOK(w, nilResponse)
	} else {
		m.respondError(w, http.StatusServiceUnavailable, "all relay_count are unavailable")
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
		"version":    config.Version,
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
	relays := make(map[BlockHashHex][]RelayEntry) // relay_count that sent the bid for a specific blockHash

	// Call the relay_count
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, relay := range m.relays {
		wg.Add(1)
		go func(relay RelayEntry) {
			defer wg.Done()
			path := fmt.Sprintf("/eth/v1/builder/header/%s/%s/%s", slot, parentHashHex, pubkey)
			url := relay.GetURI(path)
			log := log.WithField("url", url)
			responsePayload := new(types.GetHeaderResponse)
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
			if responsePayload.Data == nil || responsePayload.Data.Message == nil || responsePayload.Data.Message.Header == nil || responsePayload.Data.Message.Header.BlockHash == nilHash {
				return
			}

			blockHash := responsePayload.Data.Message.Header.BlockHash.String()
			log = log.WithFields(logrus.Fields{
				"blockNumber": responsePayload.Data.Message.Header.BlockNumber,
				"blockHash":   blockHash,
				"txRoot":      responsePayload.Data.Message.Header.TransactionsRoot.String(),
				"value":       responsePayload.Data.Message.Value.String(),
			})

			if relay.PublicKey != responsePayload.Data.Message.Pubkey {
				log.Errorf("bid pubkey mismatch. expected: %s - got: %s", relay.PublicKey.String(), responsePayload.Data.Message.Pubkey.String())
				return
			}

			// Verify the relay signature in the relay response
			ok, err := types.VerifySignature(responsePayload.Data.Message, m.builderSigningDomain, relay.PublicKey[:], responsePayload.Data.Signature[:])
			if err != nil {
				log.WithError(err).Error("error verifying relay signature")
				return
			}
			if !ok {
				log.Error("failed to verify relay signature")
				return
			}

			// Verify response coherence with proposer's input data
			responseParentHash := responsePayload.Data.Message.Header.ParentHash.String()
			if responseParentHash != parentHashHex {
				log.WithFields(logrus.Fields{
					"originalParentHash": parentHashHex,
					"responseParentHash": responseParentHash,
				}).Error("proposer and relay parent hashes are not the same")
				return
			}

			isZeroValue := responsePayload.Data.Message.Value.String() == "0"
			isEmptyListTxRoot := responsePayload.Data.Message.Header.TransactionsRoot.String() == "0x7ffe241ea60187fdb0187bfa22de35d1f9bed7ab061d9401fd47e34a54fbede1"
			if isZeroValue || isEmptyListTxRoot {
				log.Warn("ignoring bid with 0 value")
				return
			}

			log.Debug("bid received")

			mu.Lock()
			defer mu.Unlock()

			// Remember which relay_count delivered which bids (multiple relay_count might deliver the top bid)
			relays[BlockHashHex(blockHash)] = append(relays[BlockHashHex(blockHash)], relay)

			// Compare the bid with already known top bid (if any)
			if result.response.Data != nil {
				valueDiff := responsePayload.Data.Message.Value.Cmp(&result.response.Data.Message.Value)
				if valueDiff == -1 { // current bid is less profitable than already known one
					return
				} else if valueDiff == 0 { // current bid is equally profitable as already known one. Use hash as tiebreaker
					previousBidBlockHash := result.response.Data.Message.Header.BlockHash.String()
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
	result.relays = relays[BlockHashHex(result.blockHash)]
	log.WithFields(logrus.Fields{
		"blockHash":   result.blockHash,
		"blockNumber": result.response.Data.Message.Header.BlockNumber,
		"txRoot":      result.response.Data.Message.Header.TransactionsRoot.String(),
		"value":       result.response.Data.Message.Value.String(),
		"relay_count": strings.Join(RelayEntriesToStrings(result.relays), ", "),
	}).Info("best bid")

	// Remember the bid, for future logging in case of withholding
	bidKey := bidRespKey{slot: _slot, blockHash: result.blockHash}
	m.bidsLock.Lock()
	m.bids[bidKey] = result
	m.bidsLock.Unlock()

	// Return the bid
	m.respondOK(w, result.response)
}

func (m *BoostService) handleGetPayload(w http.ResponseWriter, req *http.Request) {
	log := m.log.WithFields(logrus.Fields{
		"method":  "getPayload",
		"version": config.Version,
	})

	log.Debug("getPayload")

	// Read the body first, so we can log it later on error
	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Error("could not read body of request from the beacon node")
		m.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Decode the body now
	payload := new(types.SignedBlindedBeaconBlock)
	if err := DecodeJSON(bytes.NewReader(body), payload); err != nil {
		log.WithError(err).WithField("body", string(body)).Error("could not decode request payload from the beacon-node (signed blinded beacon block)")
		m.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

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
		log.Warn("bid found but no associated relay_count")
	}

	relays := originalBid.relays
	if len(relays) == 0 {
		log.Warn("originating relay not found, sending getPayload request to all relay_count")
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
			_, err := SendHTTPRequest(requestCtx, m.httpClientGetPayload, http.MethodPost, url, ua, payload, responsePayload)

			if err != nil {
				log.WithError(err).Error("error making request to relay")
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
		log.WithField("relay_count", strings.Join(originRelays, ", ")).Error("no payload received from relay!")
		m.respondError(w, http.StatusBadGateway, errNoSuccessfulRelayResponse.Error())
		return
	}

	m.respondOK(w, result)
}

// CheckRelays sends a request to each one of the relay_count previously registered to get their status
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

			// Success: increase counter and cancel all pending requests to other relay_count
			atomic.AddUint32(&numSuccessRequestsToRelay, 1)
		}(r)
	}

	// At the end, wait for every routine and return status according to relay's ones.
	wg.Wait()
	return int(numSuccessRequestsToRelay)
}
