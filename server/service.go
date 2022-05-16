package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/flashbots/mev-boost/types"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

var (
	errInvalidSlot      = errors.New("invalid slot")
	errInvalidHash      = errors.New("invalid hash")
	errInvalidPubkey    = errors.New("invalid pubkey")
	errInvalidSignature = errors.New("invalid signature")

	errServerAlreadyRunning = errors.New("server already running")

	pathStatus            = "/eth/v1/builder/status"
	pathRegisterValidator = "/eth/v1/builder/validators"
	pathGetHeader         = "/eth/v1/builder/header/{slot:[0-9]+}/{parent_hash:0x[a-fA-F0-9]+}/{pubkey:0x[a-fA-F0-9]+}"
	// pathGetPayload        = "/eth/v1/builder/blinded_blocks"
)

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
	listenAddr string
	relays     []types.RelayEntry
	log        *logrus.Entry
	srv        *http.Server

	serverTimeouts HTTPServerTimeouts

	httpClient http.Client
}

// NewBoostService created a new BoostService
func NewBoostService(listenAddr string, relays []types.RelayEntry, log *logrus.Entry, relayRequestTimeout time.Duration) (*BoostService, error) {
	// TODO: validate relays
	if len(relays) == 0 {
		return nil, errors.New("no relays")
	}

	return &BoostService{
		listenAddr: listenAddr,
		relays:     relays,
		log:        log.WithField("module", "service"),

		serverTimeouts: NewDefaultHTTPServerTimeouts(),
		httpClient:     http.Client{Timeout: relayRequestTimeout},
	}, nil
}

func (m *BoostService) getRouter() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/", m.handleRoot)

	r.HandleFunc(pathStatus, m.handleStatus).Methods(http.MethodGet)
	r.HandleFunc(pathRegisterValidator, m.handleRegisterValidator).Methods(http.MethodPost)
	r.HandleFunc(pathGetHeader, m.handleGetHeader).Methods(http.MethodGet)
	// r.HandleFunc(pathGetPayload, m.handleGetPayload).Methods(http.MethodPost)

	r.Use(mux.CORSMethodMiddleware(r))
	loggedRouter := LoggingMiddleware(r, m.log)
	return loggedRouter
}

// StartHTTPServer starts the HTTP server for this boost service instance
func (m *BoostService) StartHTTPServer() error {
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

// RegisterValidatorV1 - returns 200 if at least one relay returns 200
func (m *BoostService) handleRegisterValidator(w http.ResponseWriter, req *http.Request) {
	log := m.log.WithField("method", "registerValidator")

	payload := new(types.SignedValidatorRegistration)
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(payload.Message.Pubkey) != 48 {
		http.Error(w, errInvalidPubkey.Error(), http.StatusBadRequest)
		return
	}

	if len(payload.Signature) != 96 {
		http.Error(w, errInvalidSignature.Error(), http.StatusBadRequest)
		return
	}

	// TODO: verify signature

	resultC := make(chan *httpResponseContainer, len(m.relays))
	for _, relay := range m.relays {
		go func(relayAddr string) {
			_url := relayAddr + pathRegisterValidator
			res, err := makeRequest(context.Background(), m.httpClient, http.MethodPost, _url, payload)
			resultC <- &httpResponseContainer{relayAddr, err, res}
		}(relay.Address)
	}

	numSuccessRequestsToRelay := 0
	for i := 0; i < cap(resultC); i++ {
		res := <-resultC
		if res.err != nil {
			log.WithFields(logrus.Fields{"error": res.err, "url": res.url}).Error("error in registerValidator to relay")
			continue
		}
		numSuccessRequestsToRelay++
	}

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

	if _, err := strconv.ParseInt(slot, 10, 64); err != nil {
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

	type safeHeaderContainer struct {
		mu             sync.Mutex
		result         *types.GetHeaderResponse
		lastRelayError error
	}
	container := safeHeaderContainer{
		result: new(types.GetHeaderResponse),
	}

	// Call the relays
	var wg sync.WaitGroup
	for _, relay := range m.relays {
		wg.Add(1)
		go func(relayAddr string) {
			defer wg.Done()
			_url := fmt.Sprintf("%s/eth/v1/builder/header/%s/%s/%s", relayAddr, slot, parentHashHex, pubkey)
			log := log.WithField("url", _url)
			res, err := makeRequest(context.Background(), m.httpClient, http.MethodGet, _url, nil)
			// Check for errors
			if err != nil {
				log.WithFields(logrus.Fields{"error": err, "url": relayAddr}).Warn("error making request to relay")
				container.mu.Lock()
				defer container.mu.Unlock()
				container.lastRelayError = err
				return
			}

			// Decode response
			responsePayload := new(types.GetHeaderResponse)
			if err := json.NewDecoder(res.Body).Decode(&responsePayload); err != nil {
				log.WithError(err).Warn("Could not unmarshal response")
				return
			}

			// Compare value of header, skip processing this result if lower fee than current
			container.mu.Lock()
			defer container.mu.Unlock()

			// Skip if invalid payload
			if responsePayload.Data == nil || responsePayload.Data.Message == nil || responsePayload.Data.Message.Header == nil || responsePayload.Data.Message.Header.BlockHash == types.NilHash {
				return
			}

			// Skip if not a higher value
			if container.result.Data != nil && responsePayload.Data.Message.Value.Cmp(&container.result.Data.Message.Value) < 1 {
				return
			}

			// Use this relay's response as mev-boost response because it's most profitable
			container.result = responsePayload
			log.WithFields(logrus.Fields{
				"blockNumber": container.result.Data.Message.Header.BlockNumber,
				"blockHash":   container.result.Data.Message.Header.BlockHash,
				"txRoot":      container.result.Data.Message.Header.TransactionsRoot.String(),
				"value":       container.result.Data.Message.Value.String(),
				"url":         relayAddr,
			}).Info("getHeader: successfully got more valuable payload header")
		}(relay.Address)
	}

	// Wait for all requests to complete...
	wg.Wait()

	if container.result.Data == nil || container.result.Data.Message.Header.BlockHash == types.NilHash {
		log.WithFields(logrus.Fields{
			"lastRelayError": container.lastRelayError,
		}).Error("getHeader: no successful response from relays")

		if container.lastRelayError != nil {
			http.Error(w, "todo", http.StatusBadGateway)
			return
		}

		http.Error(w, "no valid getHeader response", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(container.result); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// // GetPayloadV1 TODO
// func (m *BoostService) GetPayloadV1(w http.ResponseWriter, req *http.Request) {
// 	// func (m *BoostService) GetPayloadV1(ctx context.Context, block types.BlindedBeaconBlockV1, signature hexutil.Bytes) (*types.ExecutionPayloadV1, error) {
// 	method := "builder_getPayloadV1"
// 	logMethod := m.log.WithField("method", method)

// 	if len(signature) != 96 {
// 		return nil, rpcErrInvalidSignature
// 	}

// 	requestCtx, requestCtxCancel := context.WithCancel(ctx)
// 	defer requestCtxCancel()

// 	resultC := make(chan *rpcResponseContainer, len(m.relayURLs))
// 	for _, url := range m.relayURLs {
// 		go func(url string) {
// 			res, err := makeRequest(requestCtx, m.httpClient, url, "builder_getPayloadV1", []any{block, signature})
// 			resultC <- &rpcResponseContainer{url, err, res}
// 		}(url)
// 	}

// 	result := new(types.ExecutionPayloadV1)
// 	var lastRelayError error
// 	for i := 0; i < cap(resultC); i++ {
// 		res := <-resultC

// 		// Check for errors
// 		if requestCtx.Err() != nil { // request has been cancelled
// 			continue
// 		}
// 		if res.err != nil {
// 			logMethod.WithFields(logrus.Fields{"error": res.err, "url": res.url}).Error("error making request to relay")
// 			continue
// 		}
// 		if res.res.Error != nil {
// 			lastRelayError = res.res.Error
// 			logMethod.WithFields(logrus.Fields{"error": res.res.Error, "url": res.url}).Warn("error reply from relay")
// 			continue
// 		}

// 		// Decode response
// 		err := json.Unmarshal(res.res.Result, result)
// 		if err != nil {
// 			logMethod.WithFields(logrus.Fields{"error": err, "data": string(res.res.Result)}).Error("Could not unmarshal response")
// 			continue
// 		}

// 		// TODO: validate response?

// 		// Cancel other requests
// 		requestCtxCancel()
// 		logMethod.WithFields(logrus.Fields{
// 			"blockHash": result.BlockHash,
// 			"number":    result.BlockNumber,
// 			"url":       res.url,
// 		}).Info("GetPayloadV1: received payload from relay")
// 		return result, nil
// 	}

// 	logMethod.WithFields(logrus.Fields{
// 		"lastRelayError": lastRelayError,
// 	}).Error("GetPayloadV1: no valid response from relay")
// 	if lastRelayError != nil {
// 		return nil, lastRelayError
// 	}
// 	return nil, fmt.Errorf("no valid GetPayloadV1 response from relay")
// }

// // Status implements the builder_status RPC method
// func (m *BoostService) Status(w http.ResponseWriter, req *http.Request) {
// 	return &ServiceStatusOk, nil
// }
