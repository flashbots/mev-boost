package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/flashbots/mev-boost/types"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

var (
	errInvalidPubkey        = errors.New("invalid pubkey")
	errInvalidSignature     = errors.New("invalid signature")
	errServerAlreadyRunning = errors.New("server already running")

	// ServiceStatusOk indicates that the system is running as expected
	ServiceStatusOk = "OK"

	pathRegisterValidator = "/registerValidator"
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
	r.HandleFunc(pathRegisterValidator, m.handleRegisterValidator)
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
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "hello\n")
}

// RegisterValidatorV1 - returns 200 if at least one relay returns 200
func (m *BoostService) handleRegisterValidator(w http.ResponseWriter, req *http.Request) {
	method := "builder_registerValidatorV1"
	logMethod := m.log.WithField("method", method)

	payload := new(types.RegisterValidatorRequest)
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

	resultC := make(chan *httpResponseContainer, len(m.relays))
	for _, relay := range m.relays {
		go func(url string) {
			_url := url + pathRegisterValidator
			res, err := makePostRequest(context.Background(), m.httpClient, _url, payload)
			resultC <- &httpResponseContainer{url, err, res}
		}(relay.Address)
	}

	numSuccessRequestsToRelay := 0
	for i := 0; i < cap(resultC); i++ {
		res := <-resultC
		if res.err != nil {
			logMethod.WithFields(logrus.Fields{"error": res.err, "url": res.url}).Error("error in registerValidator to relay")
			continue
		}
		numSuccessRequestsToRelay++
	}

	// w.Header().Set("Content-Type", "application/json")
	if numSuccessRequestsToRelay > 0 {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadGateway)
	}
}

// // GetHeaderV1 TODO
// func (m *BoostService) GetHeaderV1(w http.ResponseWriter, req *http.Request) {
// 	// func (m *BoostService) GetHeaderV1(ctx context.Context, slot hexutil.Uint64, pubkey hexutil.Bytes, hash common.Hash) (*types.GetHeaderResponse, error) {
// 	method := "builder_getHeaderV1"
// 	logMethod := m.log.WithField("method", method)

// 	if len(pubkey) != 48 {
// 		return nil, rpcErrInvalidPubkey
// 	}

// 	type safeHeaderContainer struct {
// 		mu             sync.Mutex
// 		result         *types.GetHeaderResponse
// 		lastRelayError error
// 	}
// 	container := safeHeaderContainer{
// 		result: new(types.GetHeaderResponse),
// 	}

// 	// Call the relay
// 	var wg sync.WaitGroup
// 	for _, relayURL := range m.relayURLs {
// 		wg.Add(1)
// 		go func(url string) {
// 			defer wg.Done()
// 			res, err := makeRequest(ctx, m.httpClient, url, "builder_getHeaderV1", []interface{}{slot, pubkey, hash})

// 			// Check for errors
// 			if err != nil {
// 				logMethod.WithFields(logrus.Fields{"error": err, "url": url}).Warn("error making request to relay")
// 				return
// 			}
// 			if res.Error != nil {
// 				logMethod.WithFields(logrus.Fields{"error": res.Error, "url": url}).Warn("error reply from relay")
// 				// We can unlock using defer because we're leaving the routine right now
// 				container.mu.Lock()
// 				defer container.mu.Unlock()
// 				container.lastRelayError = res.Error
// 				return
// 			}

// 			// Decode response
// 			_result := new(types.GetHeaderResponse)
// 			err = json.Unmarshal(res.Result, _result)
// 			if err != nil {
// 				logMethod.WithFields(logrus.Fields{"error": err, "data": string(res.Result)}).Warn("Could not unmarshal response")
// 				return
// 			}

// 			// Skip processing this result if lower fee than previous
// 			container.mu.Lock()
// 			defer container.mu.Unlock()
// 			if container.result.Message.Value != nil && (_result.Message.Value == nil ||
// 				_result.Message.Value.ToInt().Cmp(container.result.Message.Value.ToInt()) < 1) {
// 				return
// 			}

// 			// Use this relay's response as mev-boost response because it's most profitable
// 			container.result = _result
// 			logMethod.WithFields(logrus.Fields{
// 				"blockNumber": container.result.Message.Header.BlockNumber,
// 				"blockHash":   container.result.Message.Header.BlockHash,
// 				"txRoot":      container.result.Message.Header.TransactionsRoot.Hex(),
// 				"value":       container.result.Message.Value.String(),
// 				"url":         url,
// 			}).Info("GetPayloadHeaderV1: successfully got more valuable payload header")
// 		}(relayURL)
// 	}

// 	// Wait for responses...
// 	wg.Wait()

// 	if container.result.Message.Header.BlockHash == types.NilHash {
// 		logMethod.WithFields(logrus.Fields{
// 			"hash":           hash,
// 			"lastRelayError": container.lastRelayError,
// 		}).Error("GetPayloadHeaderV1: no successful response from relays")

// 		if container.lastRelayError != nil {
// 			return nil, container.lastRelayError
// 		}
// 		return nil, fmt.Errorf("no valid GetHeaderV1 response from relays for hash %s", hash)
// 	}

// 	return container.result, nil
// }

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
