package lib

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/flashbots/mev-boost/lib/txroot"
	"github.com/sirupsen/logrus"
)

var httpClient = http.Client{
	Timeout: 5 * time.Second,
}

// RelayService TODO
type RelayService struct {
	relayURLs []string
	store     Store
	log       *logrus.Entry
}

func newRelayService(relayURLs []string, store Store, log *logrus.Entry) (*RelayService, error) {
	if len(relayURLs) == 0 || relayURLs[0] == "" {
		return nil, errors.New("no relayURLs")
	}

	return &RelayService{
		relayURLs: relayURLs,
		store:     store,
		log:       log.WithField("prefix", "lib/service"),
	}, nil
}

func makeRequest(ctx context.Context, url string, method string, params []interface{}) (*rpcResponse, error) {
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

	resp, err := httpClient.Do(req)
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

// ForkchoiceUpdatedV1 TODO
func (m *RelayService) ForkchoiceUpdatedV1(_ *http.Request, args *[]interface{}, result *ForkChoiceResponse) error {
	method := "relay_forkchoiceUpdatedV1"
	logMethod := m.log.WithField("method", method)

	boostPayloadID := make(hexutil.Bytes, 8)
	if _, err := rand.Read(boostPayloadID); err != nil {
		return err
	}

	var wg sync.WaitGroup
	hasValidResponse := false
	for _, url := range m.relayURLs {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			res, err := makeRequest(context.Background(), url, method, *args)

			// Check for errors
			if err != nil {
				logMethod.WithFields(logrus.Fields{"error": err, "url": url}).Error("error making request to relay")
				return
			}
			if res.Error != nil {
				logMethod.WithFields(logrus.Fields{"error": res.Error, "url": url}).Warn("error reply from relay")
				return
			}

			// Decode response
			forkchoiceResponse := new(ForkChoiceResponse)
			err = json.Unmarshal(res.Result, forkchoiceResponse)
			if err != nil {
				logMethod.WithFields(logrus.Fields{"error": err, "data": string(res.Result)}).Error("Could not unmarshal response")
				return
			}

			status := forkchoiceResponse.PayloadStatus.Status
			if status != ForkchoiceStatusValid && status != "SUCCESS" && status != "" { // SUCCESS is used by mergemock, although it's not in the engine spec (also accept empty status because mergemock)
				logMethod.WithFields(logrus.Fields{"error": err, "url": url, "status": status}).Warn("status not valid")
				return
			}

			if forkchoiceResponse.PayloadID != nil {
				m.store.SetForkchoiceResponse(boostPayloadID.String(), url, forkchoiceResponse.PayloadID.String())
				hasValidResponse = true
			}
		}(url)
	}

	wg.Wait()
	if !hasValidResponse {
		logMethod.Error("ForkchoiceUpdatedV1: no valid relay response")
		return errors.New("no valid relay response")
	}

	// Compile the response
	*result = ForkChoiceResponse{
		PayloadStatus: PayloadStatus{Status: ForkchoiceStatusValid},
		PayloadID:     &boostPayloadID,
	}

	return nil
}

// ProposeBlindedBlockV1 TODO
func (m *RelayService) ProposeBlindedBlockV1(_ *http.Request, args *SignedBlindedBeaconBlock, result *ExecutionPayloadWithTxRootV1) error {
	method := "builder_proposeBlindedBlockV1"
	logMethod := m.log.WithField("method", method)

	if args == nil || args.Message == nil {
		logMethod.Errorf("SignedBlindedBeaconBlock or SignedBlindedBeaconBlock.Message is nil: %+v", args)
		return errors.New("SignedBlindedBeaconBlock or SignedBlindedBeaconBlock.Message is nil")
	}

	var body BlindedBeaconBlockBodyPartial
	err := json.Unmarshal(args.Message.Body, &body)
	if err != nil {
		logMethod.WithField("err", err).Error("Could not unmarshal blinded body")
		return err
	}

	var blockHash string
	// Deal with allowing both camelCase and snake_case in BlindedBlock
	if body.ExecutionPayload.BlockHash != "" {
		blockHash = body.ExecutionPayload.BlockHash
	} else if body.ExecutionPayload.BlockHashCamel != "" {
		blockHash = body.ExecutionPayload.BlockHashCamel
	} else if body.ExecutionPayloadCamel.BlockHash != "" {
		blockHash = body.ExecutionPayloadCamel.BlockHash
	} else if body.ExecutionPayloadCamel.BlockHashCamel != "" {
		blockHash = body.ExecutionPayloadCamel.BlockHashCamel
	}

	payloadCached := m.store.GetExecutionPayload(common.HexToHash(blockHash))
	if payloadCached != nil {
		logMethod.WithFields(logrus.Fields{
			"blockHash": payloadCached.BlockHash,
			"number":    payloadCached.Number,
			"txRoot":    fmt.Sprintf("%#x", payloadCached.TransactionsRoot),
		}).Info("ProposeBlindedBlockV1: revealed previous payload")
		*result = *payloadCached
		return nil
	}

	requestCtx, requestCtxCancel := context.WithCancel(context.Background())
	defer requestCtxCancel()

	resultC := make(chan *rpcResponseContainer, len(m.relayURLs))
	for _, url := range m.relayURLs {
		go func(url string) {
			res, err := makeRequest(requestCtx, url, "relay_proposeBlindedBlockV1", []interface{}{args})
			resultC <- &rpcResponseContainer{url, err, res}
		}(url)
	}

	for i := 0; i < cap(resultC); i++ {
		res := <-resultC

		// Check for errors
		if requestCtx.Err() != nil { // request has been cancelled
			continue
		}
		if err != nil {
			logMethod.WithFields(logrus.Fields{"error": err, "url": res.url}).Error("error making request to relay")
			continue
		}
		if res.res.Error != nil {
			logMethod.WithFields(logrus.Fields{"error": res.res.Error, "url": res.url}).Warn("error reply from relay")
			continue
		}

		// Decode response
		err = json.Unmarshal(res.res.Result, result)
		if err != nil {
			logMethod.WithFields(logrus.Fields{"error": err, "data": string(res.res.Result)}).Error("Could not unmarshal response")
			continue
		}

		// Cancel other requests
		requestCtxCancel()
		logMethod.WithFields(logrus.Fields{
			"blockHash": result.BlockHash,
			"number":    result.Number,
			"txRoot":    fmt.Sprintf("%#x", result.TransactionsRoot),
		}).Info("ProposeBlindedBlockV1: revealed new payload from relay")
		return nil
	}

	logMethod.WithFields(logrus.Fields{
		"blockHash": blockHash,
	}).Error("ProposeBlindedBlockV1: no valid response from relay")
	return fmt.Errorf("no valid response from relay for block with hash %s", blockHash)
}

// GetPayloadHeaderV1 TODO
func (m *RelayService) GetPayloadHeaderV1(_ *http.Request, args *string, result *ExecutionPayloadWithTxRootV1) error {
	method := "engine_getPayloadV1"
	logMethod := m.log.WithField("method", method)

	payloadID := new(hexutil.Bytes)
	err := payloadID.UnmarshalText([]byte(*args))
	if err != nil {
		return err
	}

	forkchoiceResponses, found := m.store.GetForkchoiceResponse(payloadID.String())
	if !found {
		return fmt.Errorf("no ForkChoiceResponses for payloadID %s", payloadID)
	}

	// Call the relay
	resultC := make(chan *rpcResponseContainer, len(forkchoiceResponses))
	for relayURL, relayPayloadID := range forkchoiceResponses {
		go func(url, payloadID string) {
			res, err := makeRequest(context.Background(), url, "relay_getPayloadHeaderV1", []interface{}{payloadID})
			resultC <- &rpcResponseContainer{url, err, res}
		}(relayURL, relayPayloadID)
	}

	// Process the responses
	for i := 0; i < cap(resultC); i++ {
		res := <-resultC

		// Check for errors
		if res.err != nil {
			logMethod.WithFields(logrus.Fields{"error": res.err, "url": res.url}).Warn("error making request to relay")
			continue
		}
		if res.res.Error != nil {
			logMethod.WithFields(logrus.Fields{"error": res.res.Error, "url": res.url}).Warn("error reply from relay")
			continue
		}

		// Decode response
		_result := new(ExecutionPayloadWithTxRootV1)
		err := json.Unmarshal(res.res.Result, _result)
		if err != nil {
			logMethod.WithFields(logrus.Fields{"error": err, "data": string(res.res.Result)}).Warn("Could not unmarshal response")
			continue
		}

		// Skip processing this result if lower fee than previous
		if result.FeeRecipientDiff != nil {
			if _result.FeeRecipientDiff == nil || _result.FeeRecipientDiff.Cmp(result.FeeRecipientDiff) < 1 {
				continue
			}
		}

		// Use this relay's response as mev-boost response because it's most profitable
		*result = *_result

		if result.Transactions != nil {
			logMethod.WithFields(logrus.Fields{
				"blockHash": result.BlockHash,
				"number":    result.Number,
			}).Info("GetPayloadHeaderV1: calculating tx root from tx list")

			var byteTxs [][]byte
			for i, otx := range *result.Transactions {
				var tx types.Transaction
				bytesTx := common.Hex2Bytes(otx)
				if err := tx.UnmarshalBinary(bytesTx); err != nil {
					logMethod.WithFields(logrus.Fields{
						"err":   err,
						"tx":    string(bytesTx),
						"count": i,
					}).Error("Failed to decode tx")
					continue
				}
				byteTxs = append(byteTxs, bytesTx)
			}

			newRootBytes, err := txroot.TransactionsRoot(byteTxs)
			if err != nil {
				logMethod.WithField("err", err).Error("Error calculating tx root")
				continue
			}
			newRoot := common.BytesToHash(newRootBytes[:])

			if result.TransactionsRoot != nilHash {
				if newRoot != result.TransactionsRoot {
					err := fmt.Errorf("mismatched tx root: %s, %s", newRoot.String(), result.TransactionsRoot.String())
					logMethod.WithField("err", err).Error("Mismatched tx root")
					continue
				}
			}
			result.TransactionsRoot = newRoot

			// copy this payload for later retrieval in proposeBlindedBlock
			payload := new(ExecutionPayloadWithTxRootV1)
			*payload = *result
			m.store.SetExecutionPayload(result.BlockHash, payload)
		}
		result.Transactions = nil

		logMethod.WithFields(logrus.Fields{
			"blockHash": result.BlockHash,
			"number":    result.Number,
			"txRoot":    fmt.Sprintf("%#x", result.TransactionsRoot),
		}).Info("GetPayloadHeaderV1: successfully got payload header")
	}

	if result.BlockHash == nilHash {
		logMethod.WithFields(logrus.Fields{
			"payloadID": payloadID,
		}).Error("GetPayloadHeaderV1: no valid response from relay")
		return fmt.Errorf("no valid response from relay for payloadID %s", payloadID)
	}

	return nil
}
