package lib

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/flashbots/mev-middleware/lib/txroot"
	"github.com/gorilla/rpc"
	"github.com/sirupsen/logrus"
)

// MevService TODO
type MevService struct {
	executionURL string
	relayURL     string
	log          *logrus.Entry
}

// RelayService TODO
type RelayService struct {
	executionURL string
	relayURL     string
	store        Store
	log          *logrus.Entry
}

// GetPayloadArgs TODO
type GetPayloadArgs struct {
	Foo string
}

// Response TODO
type Response struct {
	Result string
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
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
func (m *MevService) ForkchoiceUpdatedV1(r *http.Request, args *[]interface{}, result *ForkChoiceResponse) error {
	method := "engine_forkchoiceUpdatedV1"
	logMethod := m.log.WithField("method", method)

	executionResp, executionErr := makeRequest(m.executionURL, method, *args)
	relayResp, relayErr := makeRequest(m.relayURL, method, *args)
	bestResponse := relayResp

	if relayErr != nil {
		logMethod.WithFields(logrus.Fields{
			"error":   relayErr,
			"url":     m.relayURL,
			"respond": string(relayResp),
			"method":  method,
		}).Warn("Could not make request to relay")

		if executionErr != nil {
			// both clients errored, abort
			logMethod.WithFields(logrus.Fields{
				"error":      executionErr,
				"relayError": relayErr,
				"url":        m.executionURL,
				"respond":    string(executionResp),
				"method":     method,
			}).Error("Could not make request to execution")
			return fmt.Errorf("relay error: %v, execution error: %v", relayErr, executionErr)
		}

		bestResponse = executionResp
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

	executionResp, executionErr := makeRequest(m.executionURL, method, []interface{}{*args})
	relayResp, relayErr := makeRequest(m.relayURL, "relay_getPayloadHeaderV1", []interface{}{*args})
	bestResponse := relayResp
	if relayErr != nil {
		logMethod.WithFields(logrus.Fields{
			"error":   relayErr,
			"url":     m.relayURL,
			"respond": string(relayResp),
			"method":  "relay_getPayloadHeaderV1",
		}).Warn("Could not make request to relay")
		if executionErr != nil {
			// both clients errored, abort
			logMethod.WithFields(logrus.Fields{
				"error":   executionErr,
				"url":     m.executionURL,
				"respond": string(executionResp),
				"method":  method,
			}).Error("Could not make request to execution")
			return fmt.Errorf("relay error: %v, execution error: %v", relayErr, executionErr)
		}

		bestResponse = executionResp
	}

	resp, err := parseRPCResponse(bestResponse)
	if err != nil {
		logMethod.WithField("err", err).Error("Could not parse response")
		return err
	}

	err = json.Unmarshal(resp.Result, result)
	if err != nil {
		resp, err = parseRPCResponse(executionResp)
		if err != nil {
			logMethod.WithField("err", err).Error("Could not parse response")
			return err
		}

		err = json.Unmarshal(resp.Result, result)
		if err != nil {
			logMethod.WithField("err", err).Error("Could not unmarshal response")
			return err
		}
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

func (m *MevService) methodNotFound(i *rpc.RequestInfo, w http.ResponseWriter) error {
	logMethod := m.log.WithField("method", i.Method)

	logMethod.Warnf("method not found, forwarding to execution client")

	req, err := http.NewRequest(http.MethodPost, m.executionURL, bytes.NewReader(i.Body))
	if err != nil {
		logMethod.WithField("err", err).Error("error creating request")
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logMethod.WithField("err", err).Error("error doing request")
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			return
		}
	}()

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)

	return err
}

func newMevService(executionURL string, relayURL string, log *logrus.Entry) (*MevService, error) {
	if executionURL == "" {
		return nil, errors.New("NewMevService must have an executionURL")
	}
	if relayURL == "" {
		return nil, errors.New("NewMevService must have an relayURL")
	}

	return &MevService{
		executionURL: executionURL,
		relayURL:     relayURL,
		log:          log.WithField("prefix", "lib/service"),
	}, nil
}

func newRelayService(executionURL string, relayURL string, store Store, log *logrus.Entry) (*RelayService, error) {
	if executionURL == "" {
		return nil, errors.New("NewRelayService must have an executionURL")
	}
	if relayURL == "" {
		return nil, errors.New("NewRelayService must have an relayURL")
	}

	return &RelayService{
		executionURL: executionURL,
		relayURL:     relayURL,
		store:        store,
		log:          log.WithField("prefix", "lib/service"),
	}, nil
}
