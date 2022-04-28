package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/mev-boost/types"
	"github.com/sirupsen/logrus"
)

var (
	defaultHTTPTimeout      = time.Second * 5
	defaultGetHeaderTimeout = time.Second * 2
)

var (
	rpcErrInvalidPubkey    = newRPCError("invalid pubkey", -32602)
	rpcErrInvalidSignature = newRPCError("invalid signature", -32602)

	// ServiceStatusOk indicates that the system is running as expected
	ServiceStatusOk = "OK"
)

// BoostService TODO
type BoostService struct {
	relayURLs []string
	log       *logrus.Entry

	httpClient      http.Client
	getHeaderClient http.Client
}

// NewBoostService created a new BoostService
func NewBoostService(relayURLs []string, log *logrus.Entry, getHeaderTimeout time.Duration) (*BoostService, error) {
	if len(relayURLs) == 0 || relayURLs[0] == "" {
		return nil, errors.New("no relayURLs")
	}

	// GetHeader timeout: fallback to default if not specified
	if getHeaderTimeout == 0 {
		getHeaderTimeout = defaultGetHeaderTimeout
	}

	return &BoostService{
		relayURLs: relayURLs,
		log:       log.WithField("prefix", "service"),

		httpClient:      http.Client{Timeout: defaultHTTPTimeout},
		getHeaderClient: http.Client{Timeout: getHeaderTimeout},
	}, nil
}

func makeRequest(ctx context.Context, client http.Client, url string, method string, params []interface{}) (*rpcResponse, error) {
	reqJSON := rpcRequest{
		ID:      "1",
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	body, err := json.Marshal(reqJSON)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseRPCResponse(respBody)
}

type rpcResponseContainer struct {
	url string
	err error
	res *rpcResponse
}

// RegisterValidatorV1 - returns OK if at least one relay returns true
func (m *BoostService) RegisterValidatorV1(ctx context.Context, message types.RegisterValidatorRequestMessage, signature hexutil.Bytes) (*string, error) {
	method := "builder_registerValidatorV1"
	logMethod := m.log.WithField("method", method)

	if len(message.Pubkey) != 48 {
		return nil, rpcErrInvalidPubkey
	}

	if len(signature) != 96 {
		return nil, rpcErrInvalidSignature
	}

	ok := false // at least one builder has returned true
	var lastRelayError error
	var wg sync.WaitGroup
	for _, url := range m.relayURLs {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			res, err := makeRequest(ctx, m.httpClient, url, method, []any{message, signature})

			// Check for errors
			if err != nil {
				logMethod.WithFields(logrus.Fields{"error": err, "url": url}).Error("error making request to relay")
				return
			}
			if res.Error != nil {
				logMethod.WithFields(logrus.Fields{"error": res.Error, "url": url}).Warn("error reply from relay")
				lastRelayError = res.Error
				return
			}

			// Decode the response
			builderResult := ""
			err = json.Unmarshal(res.Result, &builderResult)
			if err != nil {
				logMethod.WithFields(logrus.Fields{"error": err, "url": url}).Error("error unmarshalling response from relay")
				return
			}

			// Ok should be true if any one builder responds with OK
			ok = ok || builderResult == ServiceStatusOk
		}(url)
	}

	// Wait for responses...
	wg.Wait()

	// If no relay responded true, return the last error message, or a generic error
	var err error
	if !ok {
		err = lastRelayError
		if lastRelayError == nil {
			err = errors.New("no relay responded true")
		}
		return nil, err
	}
	return &ServiceStatusOk, nil
}

// GetHeaderV1 TODO
func (m *BoostService) GetHeaderV1(ctx context.Context, slot hexutil.Uint64, pubkey hexutil.Bytes, hash common.Hash) (*types.GetHeaderResponse, error) {
	method := "builder_getHeaderV1"
	logMethod := m.log.WithField("method", method)

	if len(pubkey) != 48 {
		return nil, rpcErrInvalidPubkey
	}

	// Call the relay
	result := new(types.GetHeaderResponse)
	var lastRelayError error
	var wg sync.WaitGroup
	for _, relayURL := range m.relayURLs {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			res, err := makeRequest(ctx, m.getHeaderClient, url, "builder_getHeaderV1", []interface{}{slot, pubkey, hash})

			// Check for errors
			if err != nil {
				logMethod.WithFields(logrus.Fields{"error": err, "url": url}).Warn("error making request to relay")
				return
			}
			if res.Error != nil {
				logMethod.WithFields(logrus.Fields{"error": res.Error, "url": url}).Warn("error reply from relay")
				lastRelayError = res.Error
				return
			}

			// Decode response
			_result := new(types.GetHeaderResponse)
			err = json.Unmarshal(res.Result, _result)
			if err != nil {
				logMethod.WithFields(logrus.Fields{"error": err, "data": string(res.Result)}).Warn("Could not unmarshal response")
				return
			}

			// Skip processing this result if lower fee than previous
			if result.Message.Value != nil && (_result.Message.Value == nil || _result.Message.Value.ToInt().Cmp(result.Message.Value.ToInt()) < 1) {
				return
			}

			// Use this relay's response as mev-boost response because it's most profitable
			result = _result
			logMethod.WithFields(logrus.Fields{
				"blockNumber": result.Message.Header.BlockNumber,
				"blockHash":   result.Message.Header.BlockHash,
				"txRoot":      result.Message.Header.TransactionsRoot.Hex(),
				"value":       result.Message.Value.String(),
				"url":         url,
			}).Info("GetPayloadHeaderV1: successfully got more valuable payload header")
		}(relayURL)
	}

	// Wait for responses...
	wg.Wait()

	if result.Message.Header.BlockHash == types.NilHash {
		logMethod.WithFields(logrus.Fields{
			"hash":           hash,
			"lastRelayError": lastRelayError,
		}).Error("GetPayloadHeaderV1: no successful response from relays")

		if lastRelayError != nil {
			return nil, lastRelayError
		}
		return nil, fmt.Errorf("no valid GetHeaderV1 response from relays for hash %s", hash)
	}

	return result, nil
}

// GetPayloadV1 TODO
func (m *BoostService) GetPayloadV1(ctx context.Context, block types.BlindBeaconBlockV1, signature hexutil.Bytes) (*types.ExecutionPayloadV1, error) {
	method := "builder_getPayloadV1"
	logMethod := m.log.WithField("method", method)

	if len(signature) != 96 {
		return nil, rpcErrInvalidSignature
	}

	requestCtx, requestCtxCancel := context.WithCancel(ctx)
	defer requestCtxCancel()

	resultC := make(chan *rpcResponseContainer, len(m.relayURLs))
	for _, url := range m.relayURLs {
		go func(url string) {
			res, err := makeRequest(requestCtx, m.httpClient, url, "builder_getPayloadV1", []any{block, signature})
			resultC <- &rpcResponseContainer{url, err, res}
		}(url)
	}

	result := new(types.ExecutionPayloadV1)
	var lastRelayError error
	for i := 0; i < cap(resultC); i++ {
		res := <-resultC

		// Check for errors
		if requestCtx.Err() != nil { // request has been cancelled
			continue
		}
		if res.err != nil {
			logMethod.WithFields(logrus.Fields{"error": res.err, "url": res.url}).Error("error making request to relay")
			continue
		}
		if res.res.Error != nil {
			lastRelayError = res.res.Error
			logMethod.WithFields(logrus.Fields{"error": res.res.Error, "url": res.url}).Warn("error reply from relay")
			continue
		}

		// Decode response
		err := json.Unmarshal(res.res.Result, result)
		if err != nil {
			logMethod.WithFields(logrus.Fields{"error": err, "data": string(res.res.Result)}).Error("Could not unmarshal response")
			continue
		}

		// TODO: validate response?

		// Cancel other requests
		requestCtxCancel()
		logMethod.WithFields(logrus.Fields{
			"blockHash": result.BlockHash,
			"number":    result.BlockNumber,
			"url":       res.url,
		}).Info("GetPayloadV1: received payload from relay")
		return result, nil
	}

	logMethod.WithFields(logrus.Fields{
		"lastRelayError": lastRelayError,
	}).Error("GetPayloadV1: no valid response from relay")
	if lastRelayError != nil {
		return nil, lastRelayError
	}
	return nil, fmt.Errorf("no valid GetPayloadV1 response from relay")
}

// Status implements the builder_status RPC method
func (m *BoostService) Status(ctx context.Context) (*string, error) {
	return &ServiceStatusOk, nil
}
