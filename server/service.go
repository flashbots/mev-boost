package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/go-utils/httplogger"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

var (
	errInvalidSlot      = errors.New("invalid slot")
	errInvalidHash      = errors.New("invalid hash")
	errInvalidPubkey    = errors.New("invalid pubkey")
	errInvalidSignature = errors.New("invalid signature")

	errServerAlreadyRunning = errors.New("server already running")
)

var nilHash = types.Hash{}

// HTTPServerTimeouts are various timeouts for requests to the mev-boost HTTP server
type HTTPServerTimeouts struct {
	Read       time.Duration // Timeout for body reads. None if 0.
	ReadHeader time.Duration // Timeout for header reads. None if 0.
	Write      time.Duration // Timeout for writes. None if 0.
	Idle       time.Duration // Timeout to disconnect idle client connections. None if 0.
}

// NewDefaultHTTPServerTimeouts creates default server timeouts
func NewDefaultHTTPServerTimeouts() HTTPServerTimeouts {
	return HTTPServerTimeouts{
		Read:       4 * time.Second,
		ReadHeader: 2 * time.Second,
		Write:      6 * time.Second,
		Idle:       10 * time.Second,
	}
}

// BoostService TODO
type BoostService struct {
	ctx        context.Context
	cancelFunc context.CancelFunc

	listenAddr string
	relays     []RelayEntry
	log        *logrus.Entry
	srv        *http.Server

	builderSigningDomain types.Domain
	serverTimeouts       HTTPServerTimeouts

	httpClient http.Client

	// Used by registerValidator to share new incoming registration request with the goroutine holding the ticker
	newRegistrationsRequests chan []types.SignedValidatorRegistration
	// Used by registerValidatorAtInterval to share the number of successful requests with the registerValidator handler
	numSuccessRequestsToRelay chan uint64
}

// NewBoostService created a new BoostService
func NewBoostService(listenAddr string, relays []RelayEntry, log *logrus.Entry, genesisForkVersionHex string, relayRequestTimeout time.Duration) (*BoostService, error) {
	if len(relays) == 0 {
		return nil, errors.New("no relays")
	}

	builderSigningDomain, err := ComputeDomain(types.DomainTypeAppBuilder, genesisForkVersionHex, types.Root{}.String())
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &BoostService{
		ctx:        ctx,
		cancelFunc: cancel,

		listenAddr: listenAddr,
		relays:     relays,
		log:        log.WithField("module", "service"),

		builderSigningDomain: builderSigningDomain,
		serverTimeouts:       NewDefaultHTTPServerTimeouts(),
		httpClient:           http.Client{Timeout: relayRequestTimeout},

		newRegistrationsRequests:  make(chan []types.SignedValidatorRegistration),
		numSuccessRequestsToRelay: make(chan uint64),
	}, nil
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

// StartServer starts the HTTP server for this boost service instance
func (m *BoostService) StartServer(registerValidatorInterval time.Duration) error {
	if m.srv != nil {
		return errServerAlreadyRunning
	}

	m.srv = &http.Server{
		Addr:    m.listenAddr,
		Handler: m.getRouter(),

		ReadTimeout:       m.serverTimeouts.Read,
		ReadHeaderTimeout: m.serverTimeouts.ReadHeader,
		WriteTimeout:      m.serverTimeouts.Write,
		IdleTimeout:       m.serverTimeouts.Idle,
	}

	// Start separate process to send validator preferences at regular interval.
	go m.registerValidatorAtInterval(registerValidatorInterval)
	defer m.shutdown()

	err := m.srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}

	return err
}

func (m *BoostService) handleRoot(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{}`)
}

func (m *BoostService) handleStatus(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{}`)
}

// sendValidatorPreferences is used to send the validators preferences to the registered relays
func (m *BoostService) sendValidatorPreferences(log *logrus.Entry, payload []types.SignedValidatorRegistration) uint64 {
	// We need a wait group to manage each routine used to perform the requests.
	var wg sync.WaitGroup

	// Use an atomic counter to count successful requests.
	numSuccessRequestsToRelay := uint64(0)

	// Send the validators preferences to each registered relay.
	for _, relay := range m.relays {
		wg.Add(1)

		go func(relayAddr string) {
			defer wg.Done()

			url := relayAddr + pathRegisterValidator
			log := log.WithField("url", url)

			err := SendHTTPRequest(context.Background(), m.httpClient, http.MethodPost, url, payload, nil)
			if err != nil {
				log.WithError(err).Warn("error in registerValidator to relay")
				return
			}

			atomic.AddUint64(&numSuccessRequestsToRelay, 1)
		}(relay.Address)
	}

	wg.Wait()

	return numSuccessRequestsToRelay
}

func (m *BoostService) registerValidatorAtInterval(interval time.Duration) {
	var payload []types.SignedValidatorRegistration
	log := m.log.WithField("method", "registerValidatorAtInterval")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			// mev-boost has probably stopped
			return
		case payload = <-m.newRegistrationsRequests:
			// registerValidator has received new registrations and forwards them to here
			m.numSuccessRequestsToRelay <- m.sendValidatorPreferences(log, payload)
			// Reset the timer to avoid overload
			ticker.Reset(interval)
		case <-ticker.C:
			m.sendValidatorPreferences(log, payload)
		}
	}
}

// RegisterValidatorV1 - returns 200 if at least one relay returns 200
func (m *BoostService) handleRegisterValidator(w http.ResponseWriter, req *http.Request) {
	log := m.log.WithField("method", "registerValidator")
	log.Info("registerValidator")

	payload := []types.SignedValidatorRegistration{}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, registration := range payload {
		if len(registration.Message.Pubkey) != 48 {
			http.Error(w, errInvalidPubkey.Error(), http.StatusBadRequest)
			return
		}

		if len(registration.Signature) != 96 {
			http.Error(w, errInvalidSignature.Error(), http.StatusBadRequest)
			return
		}

		ok, err := types.VerifySignature(registration.Message, m.builderSigningDomain, registration.Message.Pubkey[:], registration.Signature[:])
		if err != nil {
			log.WithError(err).WithField("registration", registration).Error("error verifying registerValidator signature")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if !ok {
			log.WithError(err).WithField("registration", registration).Error("failed to verify registerValidator signature")
			http.Error(w, errInvalidSignature.Error(), http.StatusBadRequest)
			return
		}
	}

	// Send the payload to the goroutine responsible for handling the resend at interval
	m.newRegistrationsRequests <- payload
	// Block until we get the number of successful requests back from this goroutine
	numSuccessRequestsToRelay := <-m.numSuccessRequestsToRelay

	if numSuccessRequestsToRelay > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{}`)
	} else {
		w.WriteHeader(http.StatusBadGateway)
	}
}

// GetHeaderV1 TODO
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
	log.Info("getHeader")

	if _, err := strconv.ParseUint(slot, 10, 64); err != nil {
		http.Error(w, errInvalidSlot.Error(), http.StatusBadRequest)
		return
	}

	if len(pubkey) != 98 {
		http.Error(w, errInvalidPubkey.Error(), http.StatusBadRequest)
		return
	}

	if len(parentHashHex) != 66 {
		http.Error(w, errInvalidHash.Error(), http.StatusBadRequest)
		return
	}

	result := new(types.GetHeaderResponse)
	var mu sync.Mutex

	// Call the relays
	var wg sync.WaitGroup
	for _, relay := range m.relays {
		wg.Add(1)
		go func(relayAddr string, relayPubKey types.PublicKey) {
			defer wg.Done()
			url := fmt.Sprintf("%s/eth/v1/builder/header/%s/%s/%s", relayAddr, slot, parentHashHex, pubkey)
			log := log.WithField("url", url)
			responsePayload := new(types.GetHeaderResponse)
			err := SendHTTPRequest(context.Background(), m.httpClient, http.MethodGet, url, nil, responsePayload)

			if err != nil {
				log.WithError(err).Warn("error making request to relay")
				return
			}

			// Skip if invalid payload
			if responsePayload.Data == nil || responsePayload.Data.Message == nil || responsePayload.Data.Message.Header == nil || responsePayload.Data.Message.Header.BlockHash == nilHash {
				return
			}

			log = log.WithFields(logrus.Fields{
				"blockNumber": responsePayload.Data.Message.Header.BlockNumber,
				"blockHash":   responsePayload.Data.Message.Header.BlockHash,
				"txRoot":      responsePayload.Data.Message.Header.TransactionsRoot.String(),
				"value":       responsePayload.Data.Message.Value.String(),
			})

			// Verify the relay signature in the relay response
			ok, err := types.VerifySignature(responsePayload.Data.Message, m.builderSigningDomain, relayPubKey[:], responsePayload.Data.Signature[:])
			if err != nil {
				log.WithError(err).Error("error verifying relay signature")
				return
			}
			if !ok {
				log.WithError(errInvalidSignature).Error("failed to verify relay signature")
				return
			}

			// Compare value of header, skip processing this result if lower fee than current
			mu.Lock()
			defer mu.Unlock()

			// Skip if not a higher value
			if result.Data != nil && responsePayload.Data.Message.Value.Cmp(&result.Data.Message.Value) < 1 {
				return
			}

			// Use this relay's response as mev-boost response because it's most profitable
			*result = *responsePayload
			log.Info("successfully got more valuable payload header")
		}(relay.Address, relay.PublicKey)
	}

	// Wait for all requests to complete...
	wg.Wait()

	if result.Data == nil || result.Data.Message == nil || result.Data.Message.Header == nil || result.Data.Message.Header.BlockHash == nilHash {
		log.Warn("no successful relay response")
		http.Error(w, "no successful relay response", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (m *BoostService) handleGetPayload(w http.ResponseWriter, req *http.Request) {
	log := m.log.WithField("method", "getPayload")
	log.Info("getPayload")

	payload := new(types.SignedBlindedBeaconBlock)
	if err := json.NewDecoder(req.Body).Decode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(payload.Signature) != 96 {
		http.Error(w, errInvalidSignature.Error(), http.StatusBadRequest)
		return
	}

	result := new(types.GetPayloadResponse)
	requestCtx, requestCtxCancel := context.WithCancel(context.Background())
	defer requestCtxCancel()
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, relay := range m.relays {
		wg.Add(1)
		go func(relayAddr string) {
			defer wg.Done()
			url := fmt.Sprintf("%s%s", relayAddr, pathGetPayload)
			log := log.WithField("url", url)
			responsePayload := new(types.GetPayloadResponse)
			err := SendHTTPRequest(requestCtx, m.httpClient, http.MethodPost, url, payload, responsePayload)

			if err != nil {
				log.WithError(err).Warn("error making request to relay")
				return
			}

			if responsePayload.Data == nil || responsePayload.Data.BlockHash == nilHash {
				log.Warn("invalid response")
				return
			}

			// Lock before accessing the shared payload
			mu.Lock()
			defer mu.Unlock()

			if requestCtx.Err() != nil { // request has been cancelled (or deadline exceeded)
				return
			}

			// Ensure the response blockhash matches the request
			if payload.Message.Body.ExecutionPayloadHeader.BlockHash != responsePayload.Data.BlockHash {
				log.WithFields(logrus.Fields{
					"payloadBlockHash":  payload.Message.Body.ExecutionPayloadHeader.BlockHash,
					"responseBlockHash": responsePayload.Data.BlockHash,
				}).Warn("requestBlockHash does not equal responseBlockHash")
				return
			}

			// Received successful response. Now cancel other requests and return immediately
			requestCtxCancel()
			*result = *responsePayload
			log.WithFields(logrus.Fields{
				"blockHash":   responsePayload.Data.BlockHash,
				"blockNumber": responsePayload.Data.BlockNumber,
			}).Info("getPayload: received payload from relay")
		}(relay.Address)
	}

	// Wait for all requests to complete...
	wg.Wait()

	if result.Data == nil || result.Data.BlockHash == nilHash {
		log.Warn("getPayload: no valid response from relay")
		http.Error(w, "no valid getPayload response", http.StatusBadGateway)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// CheckRelays sends a request to each one of the relays previously registered to get their status
func (m *BoostService) CheckRelays() bool {
	for _, relay := range m.relays {
		m.log.WithField("relay", relay).Info("Checking relay")

		err := SendHTTPRequest(context.Background(), m.httpClient, http.MethodGet, relay.Address+pathStatus, nil, nil)
		if err != nil {
			m.log.WithError(err).WithField("relay", relay).Error("relay check failed")
			return false
		}
	}

	return true
}

func (m *BoostService) shutdown() {
	m.cancelFunc()
}