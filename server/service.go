package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/flashbots/go-boost-utils/ssz"
	"github.com/flashbots/go-utils/httplogger"
	"github.com/flashbots/mev-boost/config"
	"github.com/flashbots/mev-boost/server/params"
	"github.com/flashbots/mev-boost/server/types"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	goacceptheaders "github.com/timewasted/go-accept-headers"
)

var (
	errNoRelays                  = errors.New("no relays")
	errInvalidSlot               = errors.New("invalid slot")
	errInvalidHash               = errors.New("invalid hash")
	errInvalidPubkey             = errors.New("invalid pubkey")
	errNoSuccessfulRelayResponse = errors.New("no successful relay response")
	errServerAlreadyRunning      = errors.New("server already running")
	errRetryWithV1API            = errors.New("relay may not support V2 API, retrying with V1 API")
)

var (
	nilHash     = phase0.Hash32{}
	nilResponse = struct{}{}
)

const relayBlacklistDuration = 30 * time.Second

type httpErrorResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type slotUID struct {
	slot phase0.Slot
	uid  uuid.UUID
}

// BoostServiceOpts provides all available options for use with NewBoostService
type BoostServiceOpts struct {
	Log                   *logrus.Entry
	ListenAddr            string
	RelayConfigs          []types.RelayConfig
	GenesisForkVersionHex string
	GenesisTime           uint64
	RelayCheck            bool
	RelayMinBid           types.U256Str

	RequestTimeoutGetHeader  time.Duration
	RequestTimeoutGetPayload time.Duration
	RequestTimeoutRegVal     time.Duration
	RequestMaxRetries        int

	TimeoutGetHeaderMs uint64
	LateInSlotTimeMs   uint64

	MetricsAddr string
}

// BoostService - the mev-boost service
type BoostService struct {
	listenAddr   string
	relayConfigs []types.RelayConfig
	log          *logrus.Entry
	srv          *http.Server
	relayCheck   bool
	relayMinBid  types.U256Str
	genesisTime  uint64

	builderSigningDomain phase0.Domain
	httpClientGetHeader  http.Client
	httpClientGetPayload http.Client
	httpClientRegVal     http.Client
	requestMaxRetries    int

	timeoutGetHeaderMs uint64
	lateInSlotTimeMs   uint64

	bids     map[string]bidResp // keeping track of bids, to log the originating relay on withholding
	bidsLock sync.Mutex

	slotUID     *slotUID
	slotUIDLock sync.Mutex

	relayConfigsLock sync.RWMutex

	metricsAddr string

	relayBlacklist     map[string]time.Time
	relayBlacklistLock sync.Mutex
}

// NewBoostService created a new BoostService
func NewBoostService(opts BoostServiceOpts) (*BoostService, error) {
	if len(opts.RelayConfigs) == 0 {
		return nil, errNoRelays
	}

	builderSigningDomain, err := ComputeDomain(ssz.DomainTypeAppBuilder, opts.GenesisForkVersionHex, phase0.Root{}.String())
	if err != nil {
		return nil, err
	}

	return &BoostService{
		listenAddr:   opts.ListenAddr,
		relayConfigs: opts.RelayConfigs,
		log:          opts.Log,
		relayCheck:   opts.RelayCheck,
		relayMinBid:  opts.RelayMinBid,
		genesisTime:  opts.GenesisTime,
		bids:         make(map[string]bidResp),
		slotUID:      &slotUID{},
		metricsAddr:  opts.MetricsAddr,

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
		requestMaxRetries:  opts.RequestMaxRetries,
		timeoutGetHeaderMs: opts.TimeoutGetHeaderMs,
		lateInSlotTimeMs:   opts.LateInSlotTimeMs,
		relayBlacklist:     make(map[string]time.Time),
	}, nil
}

func (m *BoostService) respondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set(HeaderContentType, MediaTypeJSON)
	w.WriteHeader(code)
	resp := httpErrorResp{code, message}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		m.log.WithField("response", resp).WithError(err).Error("could not write error response")
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (m *BoostService) respondOK(w http.ResponseWriter, response any) {
	w.Header().Set(HeaderContentType, MediaTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		m.log.WithField("response", response).WithError(err).Error("could not write OK response")
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (m *BoostService) getRouter() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/", m.handleRoot)

	r.HandleFunc(params.PathStatus, m.handleStatus).Methods(http.MethodGet)
	r.HandleFunc(params.PathRegisterValidator, m.handleRegisterValidator).Methods(http.MethodPost)
	r.HandleFunc(params.PathGetHeader, m.handleGetHeader).Methods(http.MethodGet)
	r.HandleFunc(params.PathGetPayload, m.handleGetPayload).Methods(http.MethodPost)
	r.HandleFunc(params.PathGetPayloadV2, m.handleGetPayloadV2).Methods(http.MethodPost)

	r.Use(mux.CORSMethodMiddleware(r))
	loggedRouter := httplogger.LoggingMiddlewareLogrus(m.log, r)
	return loggedRouter
}

func (m *BoostService) relayIsBlacklisted(relay types.RelayEntry) bool {
	if relay.URL == nil {
		return false
	}

	key := relay.URL.String()
	m.relayBlacklistLock.Lock()
	defer m.relayBlacklistLock.Unlock()

	lastFailure, ok := m.relayBlacklist[key]
	if !ok {
		return false
	}

	if time.Since(lastFailure) > relayBlacklistDuration {
		delete(m.relayBlacklist, key)
		return false
	}

	return true
}

func (m *BoostService) markRelayFailure(relay types.RelayEntry) {
	if relay.URL == nil {
		return
	}

	m.relayBlacklistLock.Lock()
	m.relayBlacklist[relay.URL.String()] = time.Now()
	m.relayBlacklistLock.Unlock()
}

func (m *BoostService) clearRelayFailure(relay types.RelayEntry) {
	if relay.URL == nil {
		return
	}

	m.relayBlacklistLock.Lock()
	delete(m.relayBlacklist, relay.URL.String())
	m.relayBlacklistLock.Unlock()
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

// StartMetricsServer starts the HTTP server for exporting metrics
func (m *BoostService) StartMetricsServer() error {
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		metrics.WritePrometheus(w, true)
	})
	return http.ListenAndServe(m.metricsAddr, serveMux)
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

func (m *BoostService) handleRoot(w http.ResponseWriter, _ *http.Request) {
	m.respondOK(w, nilResponse)
}

// handleStatus sends calls to the status endpoint of every relay.
// It returns OK if at least one returned OK, and returns error otherwise.
func (m *BoostService) handleStatus(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set(HeaderKeyVersion, config.Version)
	if !m.relayCheck || m.CheckRelays() > 0 {
		IncrementBeaconNodeStatus(strconv.Itoa(http.StatusOK), params.PathStatus)
		m.respondOK(w, nilResponse)
	} else {
		IncrementBeaconNodeStatus(strconv.Itoa(http.StatusServiceUnavailable), params.PathStatus)
		m.respondError(w, http.StatusServiceUnavailable, "all relays are unavailable")
	}
}

// handleRegisterValidator returns StatusOK if at least one relay returns StatusOK, else StatusBadGateway.
// This forwards the message from the node to relays with minimal overhead. The registrations will maintain their
// original encoding (SSZ or JSON) from the node.
func (m *BoostService) handleRegisterValidator(w http.ResponseWriter, req *http.Request) {
	log := m.log.WithField("method", "registerValidator")
	log.Debug("handling request")

	// Get the user agent
	ua := UserAgent(req.Header.Get(HeaderUserAgent))
	log = log.WithFields(logrus.Fields{"ua": ua})

	// Additional header fields
	header := req.Header
	header.Set(HeaderUserAgent, wrapUserAgent(ua))
	header.Set(HeaderDateMilliseconds, fmt.Sprintf("%d", time.Now().UTC().UnixMilli()))

	// Read the validator registrations
	regBytes, err := io.ReadAll(req.Body)
	if err != nil {
		IncrementBeaconNodeStatus(strconv.Itoa(http.StatusInternalServerError), params.PathRegisterValidator)
		m.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	req.Body.Close()

	// Send the registrations to each relay
	err = m.registerValidator(log, regBytes, header)
	if err == nil {
		// One of the relays responded OK
		IncrementBeaconNodeStatus(strconv.Itoa(http.StatusOK), params.PathRegisterValidator)
		m.respondOK(w, nilResponse)
	} else {
		// None of the relays responded OK
		IncrementBeaconNodeStatus(strconv.Itoa(http.StatusBadGateway), params.PathRegisterValidator)
		m.respondError(w, http.StatusBadGateway, err.Error())
	}
}

// handleGetHeader requests bids from the relays
func (m *BoostService) handleGetHeader(w http.ResponseWriter, req *http.Request) {
	var (
		vars          = mux.Vars(req)
		parentHashHex = vars["parent_hash"]
		pubkey        = vars["pubkey"]
		ua            = UserAgent(req.Header.Get(HeaderUserAgent))

		rawProposerAcceptContentTypes    = req.Header.Get(HeaderAccept)
		parsedProposerAcceptContentTypes = goacceptheaders.Parse(rawProposerAcceptContentTypes)
		headerTimeoutString              = req.Header.Get(HeaderTimeoutMs)
	)

	// Parse the slot
	slotValue, err := strconv.ParseUint(vars["slot"], 10, 64)
	if err != nil {
		IncrementBeaconNodeStatus(strconv.Itoa(http.StatusBadRequest), params.PathGetHeader)
		m.respondError(w, http.StatusBadRequest, errInvalidSlot.Error())
		return
	}
	slot := phase0.Slot(slotValue)

	// Add relevant fields to the logger
	log := m.log.WithFields(logrus.Fields{
		"method":                        "getHeader",
		"slot":                          slot,
		"parentHash":                    parentHashHex,
		"pubkey":                        pubkey,
		"ua":                            ua,
		"rawProposerAcceptContentTypes": rawProposerAcceptContentTypes,
	})
	log.Debug("handling request")

	if headerTimeoutString == "" {
		headerTimeoutString = "0"
	}

	headerTimeout, err := strconv.ParseUint(headerTimeoutString, 10, 64)
	if err != nil {
		m.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	// Query the relays for the header
	result, err := m.getHeader(log, slot, pubkey, parentHashHex, ua, rawProposerAcceptContentTypes, headerTimeout)
	if err != nil {
		IncrementBeaconNodeStatus(strconv.Itoa(http.StatusBadRequest), params.PathGetHeader)
		m.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Bail if none of the relays returned a bid
	if result.response.IsEmpty() {
		log.Info("no bid received")
		IncrementBeaconNodeStatus(strconv.Itoa(http.StatusNoContent), params.PathGetHeader)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Remember the bid, for future logging in case of withholding
	m.bidsLock.Lock()
	m.bids[bidKey(slot, result.bidInfo.blockHash)] = result
	m.bidsLock.Unlock()

	// Log result
	valueEth := weiBigIntToEthBigFloat(result.bidInfo.value.ToBig())
	log.WithFields(logrus.Fields{
		"blockHash":   result.bidInfo.blockHash.String(),
		"blockNumber": result.bidInfo.blockNumber,
		"txRoot":      result.bidInfo.txRoot.String(),
		"value":       valueEth.Text('f', 18),
		"relays":      strings.Join(types.RelayEntriesToStrings(result.relays), ", "),
	}).Info("best bid")

	// Default to JSON if the proposer provides nothing
	if len(parsedProposerAcceptContentTypes) == 0 {
		log.Info("no proposer accepts, defaulting to JSON")
		parsedProposerAcceptContentTypes = goacceptheaders.AcceptSlice{{Type: MediaTypeJSON}}
	}

	// Get the proposer's highest quality acceptable media type
	proposerPreferredContentType, err := parsedProposerAcceptContentTypes.Negotiate(MediaTypeJSON, MediaTypeOctetStream)
	if err != nil {
		log.WithError(err).Warn("failed to negotiate preferred content-type")
		proposerPreferredContentType = MediaTypeJSON
	}

	// Respond appropriately
	switch proposerPreferredContentType {
	case MediaTypeJSON:
		IncrementBeaconNodeStatus(strconv.Itoa(http.StatusOK), params.PathGetHeader)
		log.Debug("responding with JSON")
		m.respondGetHeaderJSON(w, &result)

	case MediaTypeOctetStream:
		IncrementBeaconNodeStatus(strconv.Itoa(http.StatusOK), params.PathGetHeader)
		log.Debug("responding with SSZ")
		m.respondGetHeaderSSZ(w, &result)
	default:
		IncrementBeaconNodeStatus(strconv.Itoa(http.StatusNotAcceptable), params.PathGetHeader)
		message := fmt.Sprintf("unsupported media type: %s", proposerPreferredContentType)
		log.Error(message)
		m.respondError(w, http.StatusNotAcceptable, message)
	}
}

// Deprecated: For reference: https://github.com/ethereum/builder-specs/issues/119
// handleGetPayload requests the payload from the relays
func (m *BoostService) handleGetPayload(w http.ResponseWriter, req *http.Request) {
	var (
		userAgent                        = wrapUserAgent(UserAgent(req.Header.Get(HeaderUserAgent)))
		proposerContentType              = req.Header.Get(HeaderContentType)
		proposerAcceptContentTypes       = req.Header.Get(HeaderAccept)
		parsedProposerAcceptContentTypes = goacceptheaders.Parse(proposerAcceptContentTypes)
		proposerEthConsensusVersion      = req.Header.Get(HeaderEthConsensusVersion)
	)

	// Do the initial debug log
	log := m.log.WithFields(logrus.Fields{
		"method":                      "getPayload",
		"userAgent":                   userAgent,
		"proposerContentType":         proposerContentType,
		"proposerAcceptContentTypes":  proposerAcceptContentTypes,
		"proposerEthConsensusVersion": proposerEthConsensusVersion,
	})
	log.Debug("handling request")

	// Read the body first, so we can log it later on error
	signedBlindedBlockBytes, err := io.ReadAll(req.Body)
	if err != nil {
		IncrementBeaconNodeStatus(strconv.Itoa(http.StatusBadRequest), params.PathGetPayload)
		log.WithError(err).Error("could not read body of request from the beacon node")
		m.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Query the relays for the payload
	result, originalBid := m.getPayload(log, signedBlindedBlockBytes, userAgent, proposerContentType, proposerAcceptContentTypes, proposerEthConsensusVersion)

	// If no payload has been received from relay, log loudly about withholding!
	if result.response == nil || getPayloadResponseIsEmpty(result.response) {
		IncrementBeaconNodeStatus(strconv.Itoa(http.StatusBadGateway), params.PathGetPayload)
		originRelays := types.RelayEntriesToStrings(originalBid.relays)
		log.WithField("relaysWithBid", strings.Join(originRelays, ", ")).Error("no payload received from relay!")
		m.respondError(w, http.StatusBadGateway, errNoSuccessfulRelayResponse.Error())
		return
	}

	// Default to JSON if the proposer provides nothing
	if len(parsedProposerAcceptContentTypes) == 0 {
		log.Info("no proposer accepts, defaulting to JSON")
		parsedProposerAcceptContentTypes = goacceptheaders.AcceptSlice{{Type: MediaTypeJSON}}
	}

	// Get the proposer's highest quality acceptable media type
	proposerPreferredContentType, err := parsedProposerAcceptContentTypes.Negotiate(MediaTypeJSON, MediaTypeOctetStream)
	if err != nil {
		log.WithError(err).Warn("failed to negotiate preferred content-type")
		proposerPreferredContentType = MediaTypeJSON
	}

	// Respond appropriately
	switch proposerPreferredContentType {
	case MediaTypeJSON:
		IncrementBeaconNodeStatus(strconv.Itoa(http.StatusOK), params.PathGetPayload)
		log.Debug("responding with JSON")
		m.respondGetPayloadJSON(w, result.response)
	case MediaTypeOctetStream:
		IncrementBeaconNodeStatus(strconv.Itoa(http.StatusOK), params.PathGetPayload)
		log.Debug("responding with SSZ")
		m.respondGetPayloadSSZ(w, result.response)
	default:
		IncrementBeaconNodeStatus(strconv.Itoa(http.StatusNotAcceptable), params.PathGetPayload)
		message := fmt.Sprintf("unsupported media type: %s", proposerPreferredContentType)
		log.Error(message)
		m.respondError(w, http.StatusNotAcceptable, message)
	}
}

// handleGetPayloadV2 requests the payload submission from the relays but does not return the execution payload and blobs
func (m *BoostService) handleGetPayloadV2(w http.ResponseWriter, req *http.Request) {
	var (
		userAgent                   = wrapUserAgent(UserAgent(req.Header.Get(HeaderUserAgent)))
		proposerContentType         = req.Header.Get(HeaderContentType)
		proposerEthConsensusVersion = req.Header.Get(HeaderEthConsensusVersion)
		acceptContentType           = MediaTypeJSON
	)

	// Do the initial debug log
	log := m.log.WithFields(logrus.Fields{
		"method":                      "handleGetPayloadV2",
		"userAgent":                   userAgent,
		"proposerContentType":         proposerContentType,
		"proposerEthConsensusVersion": proposerEthConsensusVersion,
	})
	log.Debug("handling request")

	// Read the body first, so we can log it later on error
	signedBlindedBlockBytes, err := io.ReadAll(req.Body)
	if err != nil {
		IncrementBeaconNodeStatus(strconv.Itoa(http.StatusBadRequest), params.PathGetPayloadV2)
		log.WithError(err).Error("could not read body of request from the beacon node")
		m.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Submit the signed blinded beacon block to relays
	result, originalBid := m.getPayloadV2(log, signedBlindedBlockBytes, userAgent, proposerContentType, acceptContentType, proposerEthConsensusVersion)

	// If no relay accepted the submission, log about the failure
	if !result.success {
		IncrementBeaconNodeStatus(strconv.Itoa(http.StatusBadGateway), params.PathGetPayloadV2)
		originRelays := types.RelayEntriesToStrings(originalBid.relays)
		log.WithField("relaysWithBid", strings.Join(originRelays, ", ")).Error("no relay accepted the signed blinded beacon block submission!")
		m.respondError(w, http.StatusBadGateway, errNoSuccessfulRelayResponse.Error())
		return
	}

	IncrementBeaconNodeStatus(strconv.Itoa(http.StatusAccepted), params.PathGetPayloadV2)
	log.Info("successfully submitted signed blinded beacon block to relay")
	w.WriteHeader(http.StatusAccepted)
}

// CheckRelays sends a request to each one of the relays previously registered to get their status
func (m *BoostService) CheckRelays() int {
	var wg sync.WaitGroup
	var numSuccessRequestsToRelay uint32

	m.relayConfigsLock.RLock()
	relayConfigs := m.relayConfigs
	m.relayConfigsLock.RUnlock()

	for _, relayConfig := range relayConfigs {
		wg.Add(1)

		go func(relay types.RelayEntry) {
			defer wg.Done()
			url := relay.GetURI(params.PathStatus)
			log := m.log.WithField("url", url)
			log.Debug("checking relay status")

			start := time.Now()
			code, err := SendHTTPRequest(context.Background(), m.httpClientGetHeader, http.MethodGet, url, "", nil, nil, nil)
			RecordRelayLatency(params.PathStatus, relay.URL.Hostname(), float64(time.Since(start).Milliseconds()))
			if err != nil {
				log.WithError(err).Error("relay status error - request failed")
				return
			}
			RecordRelayStatusCode(strconv.Itoa(code), params.PathStatus, relay.URL.Hostname())
			if code == http.StatusOK {
				log.Debug("relay status OK")
			} else {
				log.Errorf("relay status error - unexpected status code %d", code)
				return
			}

			// Success: increase counter and cancel all pending requests to other relays
			atomic.AddUint32(&numSuccessRequestsToRelay, 1)
		}(relayConfig.RelayEntry)
	}

	// At the end, wait for every routine and return status according to relay's ones.
	wg.Wait()
	return int(numSuccessRequestsToRelay)
}

// UpdateConfig updates the relay configs and timeout settings
func (m *BoostService) UpdateConfig(relayConfigs []types.RelayConfig, timeoutGetHeaderMs, lateInSlotTimeMs uint64) {
	m.relayConfigsLock.Lock()
	defer m.relayConfigsLock.Unlock()

	m.relayConfigs = relayConfigs
	m.timeoutGetHeaderMs = timeoutGetHeaderMs
	m.lateInSlotTimeMs = lateInSlotTimeMs
}
