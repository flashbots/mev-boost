package lib

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/flashbots/mev-middleware/lib/txroot"
	"github.com/sirupsen/logrus"
)

// RelayService TODO
type RelayService struct {
	relayURL string
	store    Store
	log      *logrus.Entry
}

func newRelayService(relayURL string, store Store, log *logrus.Entry) (*RelayService, error) {
	if relayURL == "" {
		return nil, errors.New("NewRelayService must have an relayURL")
	}

	return &RelayService{
		relayURL: relayURL,
		store:    store,
		log:      log.WithField("prefix", "lib/service"),
	}, nil
}

// GetPayloadArgs TODO
type GetPayloadArgs struct {
	Foo string
}

// Response TODO
type Response struct {
	Result string
}

func makeRequest(url string, method string, params []interface{}) ([]byte, error) {
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

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(resp.Body)
}

// ForkchoiceUpdatedV1 TODO
func (m *RelayService) ForkchoiceUpdatedV1(r *http.Request, args *[]interface{}, result *ForkChoiceResponse) error {
	method := "engine_forkchoiceUpdatedV1"
	logMethod := m.log.WithField("method", method)

	relayResp, relayErr := makeRequest(m.relayURL, method, *args)
	bestResponse := relayResp

	if relayErr != nil {
		logMethod.WithFields(logrus.Fields{
			"error":   relayErr,
			"url":     m.relayURL,
			"respond": string(relayResp),
			"method":  method,
		}).Warn("Could not make request to relay")
		return fmt.Errorf("relay error: %v", relayErr)
	}
	resp, err := parseRPCResponse(bestResponse)
	if err != nil {
		logMethod.WithField("err", err).Error("Could not parse response")
		return err
	}

	err = json.Unmarshal(resp.Result, result)
	if err != nil {
		logMethod.WithField("err", err).Error("Could not unmarshal response", method, err)
		return err
	}

	return nil
}

// ProposeBlindedBlockV1 TODO
func (m *RelayService) ProposeBlindedBlockV1(r *http.Request, args *SignedBlindedBeaconBlock, result *ExecutionPayloadWithTxRootV1) error {
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

	payloadCached := m.store.Get(common.HexToHash(blockHash))
	if payloadCached != nil {
		logMethod.WithFields(logrus.Fields{
			"blockHash": payloadCached.BlockHash,
			"number":    payloadCached.Number,
			"txRoot":    fmt.Sprintf("%#x", payloadCached.TransactionsRoot),
		}).Info("ProposeBlindedBlockV1: revealed previous payload from execution client")
		*result = *payloadCached
		return nil
	}

	relayResp, err := makeRequest(m.relayURL, "relay_proposeBlindedBlockV1", []interface{}{args})
	if err != nil {
		logMethod.WithFields(logrus.Fields{
			"error":  err,
			"url":    m.relayURL,
			"method": "relay_proposeBlindedBlockV1",
		}).Error("Could not make request to relay")
		return err
	}

	resp, err := parseRPCResponse(relayResp)
	if err != nil {
		logMethod.WithField("err", err).Error("Could not parse response")
		return err
	}

	err = json.Unmarshal(resp.Result, result)
	if err != nil {
		logMethod.WithField("err", err).Error("Could not unmarshal response")
		return err
	}

	logMethod.WithFields(logrus.Fields{
		"blockHash": result.BlockHash,
		"number":    result.Number,
		"txRoot":    fmt.Sprintf("%#x", result.TransactionsRoot),
	}).Info("ProposeBlindedBlockV1: revealed new payload from relay")
	return nil
}

var nilHash = common.Hash{}

// GetPayloadHeaderV1 TODO
func (m *RelayService) GetPayloadHeaderV1(r *http.Request, args *string, result *ExecutionPayloadWithTxRootV1) error {
	method := "engine_getPayloadV1"
	logMethod := m.log.WithField("method", method)

	relayResp, relayErr := makeRequest(m.relayURL, "relay_getPayloadHeaderV1", []interface{}{*args})
	bestResponse := relayResp
	if relayErr != nil {
		logMethod.WithFields(logrus.Fields{
			"error":   relayErr,
			"url":     m.relayURL,
			"respond": string(relayResp),
			"method":  "relay_getPayloadHeaderV1",
		}).Warn("Could not make request to relay")
		return fmt.Errorf("relay error: %v", relayErr)
	}

	resp, err := parseRPCResponse(bestResponse)
	if err != nil {
		logMethod.WithField("err", err).Error("Could not parse response")
		return err
	}

	err = json.Unmarshal(resp.Result, result)
	if err != nil {
		logMethod.WithField("err", err).Error("Could not unmarshal response")
		return err
	}

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
				return fmt.Errorf("failed to decode tx %d: %v", i, err)
			}
			byteTxs = append(byteTxs, bytesTx)
		}

		newRootBytes, err := txroot.TransactionsRoot(byteTxs)
		if err != nil {
			logMethod.WithField("err", err).Error("Error calculating tx root")
			return err
		}
		newRoot := common.BytesToHash(newRootBytes[:])

		if result.TransactionsRoot != nilHash {
			if newRoot != result.TransactionsRoot {
				err := fmt.Errorf("mismatched tx root: %s, %s", newRoot.String(), result.TransactionsRoot.String())
				logMethod.WithField("err", err).Error("Mismatched tx root")
				return err
			}
		}
		result.TransactionsRoot = newRoot

		// copy this payload for later retrieval in proposeBlindedBlock
		payload := new(ExecutionPayloadWithTxRootV1)
		*payload = *result
		m.store.Set(result.BlockHash, payload)
	}
	result.Transactions = nil

	logMethod.WithFields(logrus.Fields{
		"blockHash": result.BlockHash,
		"number":    result.Number,
		"txRoot":    fmt.Sprintf("%#x", result.TransactionsRoot),
	}).Info("GetPayloadHeaderV1: successfully got payload header")
	return nil
}
