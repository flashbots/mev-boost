package lib

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

	"github.com/gorilla/rpc"
	"github.com/sirupsen/logrus"
)

var (
	defaultHTTPTimeout      = time.Second * 5
	defaultGetHeaderTimeout = time.Second * 2
)

// BoostService TODO
type BoostService struct {
	relayURLs []string
	log       *logrus.Entry

	httpClient      http.Client
	getHeaderClient http.Client
}

func newBoostService(relayURLs []string, log *logrus.Entry) (*BoostService, error) {
	if len(relayURLs) == 0 || relayURLs[0] == "" {
		return nil, errors.New("no relayURLs")
	}

	return &BoostService{
		relayURLs: relayURLs,
		log:       log.WithField("prefix", "lib/service"),

		httpClient:      http.Client{Timeout: defaultHTTPTimeout},
		getHeaderClient: http.Client{Timeout: defaultGetHeaderTimeout},
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

// SetFeeRecipientV1 - returns true if at least one relay returns true
func (m *BoostService) SetFeeRecipientV1(_ *http.Request, args *[]interface{}, result *bool) error {
	method := "builder_setFeeRecipientV1"
	logMethod := m.log.WithField("method", method)
	*result = false

	if len(*args) != 4 {
		return fmt.Errorf("invalid number of params: %d", len(*args))
	}

	var lastRelayError error
	var wg sync.WaitGroup
	for _, url := range m.relayURLs {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			res, err := makeRequest(context.Background(), m.httpClient, url, method, *args)

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
			_result := false
			err = json.Unmarshal(res.Result, &_result)
			if err != nil {
				logMethod.WithFields(logrus.Fields{"error": err, "url": url}).Error("error unmarshalling response from relay")
				return
			}

			// Result should be true if any one relay responds true
			*result = *result || _result
		}(url)
	}

	// Wait for responses...
	wg.Wait()

	// If no relay responded true, return the last error message, or a generic error
	var err error
	if !*result {
		err = lastRelayError
		if lastRelayError == nil {
			err = errors.New("no relay responded true")
		}
	}
	return err
}

// GetHeaderV1 TODO
func (m *BoostService) GetHeaderV1(_ *http.Request, blockHash *string, result *GetHeaderResponse) error {
	method := "builder_getHeaderV1"
	logMethod := m.log.WithField("method", method)

	if len(*blockHash) != 66 {
		return fmt.Errorf("invalid block hash: %s", *blockHash)
	}

	// Call the relay
	var lastRelayError error
	var wg sync.WaitGroup
	for _, relayURL := range m.relayURLs {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			res, err := makeRequest(context.Background(), m.getHeaderClient, url, "builder_getHeaderV1", []interface{}{*blockHash})

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
			_result := new(GetHeaderResponse)
			err = json.Unmarshal(res.Result, _result)
			if err != nil {
				logMethod.WithFields(logrus.Fields{"error": err, "data": string(res.Result)}).Warn("Could not unmarshal response")
				return
			}

			// Skip processing this result if lower fee than previous
			if result.Value != nil && (_result.Value == nil || _result.Value.Cmp(result.Value) < 1) {
				return
			}

			// Use this relay's response as mev-boost response because it's most profitable
			*result = *_result
			logMethod.WithFields(logrus.Fields{
				"blockNumber": result.Header.BlockNumber,
				"blockHash":   result.Header.BlockHash,
				"txRoot":      result.Header.TransactionsRoot.Hex(),
				"value":       result.Value.String(),
				"url":         url,
			}).Info("GetPayloadHeaderV1: successfully got more valuable payload header")
		}(relayURL)
	}

	// Wait for responses...
	wg.Wait()

	if result.Header.BlockHash == nilHash {
		logMethod.WithFields(logrus.Fields{
			"hash":           *blockHash,
			"lastRelayError": lastRelayError,
		}).Error("GetPayloadHeaderV1: no successful response from relays")

		if lastRelayError != nil {
			return lastRelayError
		}
		return fmt.Errorf("no valid GetHeaderV1 response from relays for hash %s", *blockHash)
	}

	return nil
}

// GetPayloadV1 TODO
func (m *BoostService) GetPayloadV1(_ *http.Request, args *[]interface{}, result *ExecutionPayloadV1) error {
	method := "builder_getPayloadV1"
	logMethod := m.log.WithField("method", method)

	if args == nil || len(*args) != 2 {
		logMethod.Errorf("invalid params: %+v", args)
		return errors.New("invalid params")
	}

	requestCtx, requestCtxCancel := context.WithCancel(context.Background())
	defer requestCtxCancel()

	resultC := make(chan *rpcResponseContainer, len(m.relayURLs))
	for _, url := range m.relayURLs {
		go func(url string) {
			res, err := makeRequest(requestCtx, m.httpClient, url, "builder_getPayloadV1", *args)
			resultC <- &rpcResponseContainer{url, err, res}
		}(url)
	}

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
		return nil
	}

	logMethod.WithFields(logrus.Fields{
		"lastRelayError": lastRelayError,
	}).Error("GetPayloadV1: no valid response from relay")
	if lastRelayError != nil {
		return lastRelayError
	}
	return fmt.Errorf("no valid GetPayloadV1 response from relay")
}

func (m *BoostService) methodNotFound(i *rpc.RequestInfo, w http.ResponseWriter) error {
	// logMethod := m.log.WithField("method", i.Method)
	return fmt.Errorf("method %s not found", i.Method)
}
