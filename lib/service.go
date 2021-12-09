package lib

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/gorilla/rpc"
)

// MevService TODO
type MevService struct {
	executionURL string
	relayURL     string
}

// RelayService TODO
type RelayService struct {
	executionURL string
	relayURL     string
	payloadMap   map[common.Hash]*ExecutionPayloadWithTxRootV1 // map stateRoot to ExecutionPayloadWithTxRootV1. TODO: this has issues, in that stateRoot could actually be the same between different payloads
	// TODO: clean this up periodically
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
func (m *MevService) ForkchoiceUpdatedV1(r *http.Request, args *[]interface{}, result *interface{}) error {
	executionResp, executionErr := makeRequest(m.executionURL, "engine_forkchoiceUpdatedV1", *args)
	relayResp, relayErr := makeRequest(m.relayURL, "engine_forkchoiceUpdatedV1", *args)

	bestResponse := relayResp
	if relayErr != nil {
		log.Print("error in relay resp", relayErr, string(relayResp))
		if executionErr != nil {
			// both clients errored, abort
			log.Print("error in both resp", executionResp, string(executionResp))
			return relayErr
		}

		bestResponse = executionResp
	}
	resp, err := parseRPCResponse(bestResponse)
	if err != nil {
		return err
	}

	err = json.Unmarshal(resp.Result, result)
	if err != nil {
		log.Print("error unmarshaling result", err)
		return err
	}

	return nil
}

// ExecutePayloadV1 TODO
func (m *MevService) ExecutePayloadV1(r *http.Request, args *ExecutionPayloadWithTxRootV1, result *catalyst.ExecutePayloadResponse) error {
	executionResp, executionErr := makeRequest(m.executionURL, "engine_executePayloadV1", []interface{}{args})
	relayResp, relayErr := makeRequest(m.relayURL, "engine_executePayloadV1", []interface{}{args})

	bestResponse := relayResp
	if relayErr != nil {
		log.Print("error in relay resp", relayErr, string(relayResp))
		if executionErr != nil {
			// both clients errored, abort
			log.Print("error in both resp", executionResp, string(executionResp))
			return relayErr
		}

		bestResponse = executionResp
	}
	resp, err := parseRPCResponse(bestResponse)
	if err != nil {
		return err
	}

	err = json.Unmarshal(resp.Result, result)
	if err != nil {
		log.Print("error unmarshaling result", err)
		return err
	}

	return nil
}

// ProposeBlindedBlockV1 TODO
func (m *RelayService) ProposeBlindedBlockV1(r *http.Request, args *SignedBlindedBeaconBlock, result *ExecutionPayloadWithTxRootV1) error {
	payload, ok := m.payloadMap[common.HexToHash(args.Message.StateRoot)]
	if ok {
		log.Println("proposed previous block from execution client", payload.BlockHash, payload.Number)
		*result = *payload
		return nil
	}
	relayResp, relayErr := makeRequest(m.relayURL, "builder_proposeBlindedBlockV1", []interface{}{args})
	if relayErr != nil {
		return relayErr
	}

	resp, err := parseRPCResponse(relayResp)
	if err != nil {
		return err
	}

	err = json.Unmarshal(resp.Result, result)
	if err != nil {
		log.Print("error unmarshaling result", err)
		return err
	}

	log.Println("proposed from relay", result.BlockHash, result.Number)
	return nil
}

var nilHash = common.Hash{}

// GetPayloadHeaderV1 TODO
func (m *RelayService) GetPayloadHeaderV1(r *http.Request, args *string, result *ExecutionPayloadWithTxRootV1) error {
	executionResp, executionErr := makeRequest(m.executionURL, "engine_getPayloadV1", []interface{}{*args})
	relayResp, relayErr := makeRequest(m.relayURL, "engine_getPayloadV1", []interface{}{*args})

	bestResponse := relayResp
	if relayErr != nil {
		log.Print("error in relay resp", relayErr, string(relayResp))
		if executionErr != nil {
			// both clients errored, abort
			log.Print("error in both resp", executionResp, string(executionResp))
			return relayErr
		}

		bestResponse = executionResp
	}
	resp, err := parseRPCResponse(bestResponse)
	if err != nil {
		return err
	}

	err = json.Unmarshal(resp.Result, result)
	if err != nil {
		log.Print("error unmarshaling result", err)
		return err
	}
	result.Transactions = nil

	if result.TransactionsRoot == nilHash {
		// copy this payload for later retrieval in proposeBlindedBlock
		m.payloadMap[result.StateRoot] = new(ExecutionPayloadWithTxRootV1)
		*m.payloadMap[result.StateRoot] = *result

		txs := types.Transactions{}
		for i, otx := range result.Transactions {
			var tx types.Transaction
			if err := tx.UnmarshalBinary(otx); err != nil {
				return fmt.Errorf("failed to decode tx %d: %v", i, err)
			}
			txs = append(txs, &tx)
		}
		result.TransactionsRoot = types.DeriveSha(txs, trie.NewStackTrie(nil))
	}

	log.Println("got payload header", result.BlockHash, result.Number)
	return nil
}

func (m *MevService) methodNotFound(i *rpc.RequestInfo, w http.ResponseWriter) error {
	log.Print("method not found, forwarding to execution client: ", i.Method)

	req, err := http.NewRequest(http.MethodPost, m.executionURL, bytes.NewReader(i.Body))
	if err != nil {
		log.Print("error in method not found: creating request", i.Method, err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Print("error in method not found: doing request", i.Method, err)
		return err
	}
	defer resp.Body.Close()

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)

	return err
}

func newMevService(executionURL string, relayURL string) (*MevService, error) {
	if executionURL == "" {
		return nil, errors.New("NewRouter must have an executionURL")
	}
	if relayURL == "" {
		return nil, errors.New("NewRouter must have an relayURL")
	}

	return &MevService{
		executionURL: executionURL,
		relayURL:     relayURL,
	}, nil
}

func newRelayService(executionURL string, relayURL string) (*RelayService, error) {
	if executionURL == "" {
		return nil, errors.New("NewRouter must have an executionURL")
	}
	if relayURL == "" {
		return nil, errors.New("NewRouter must have an relayURL")
	}

	return &RelayService{
		executionURL: executionURL,
		relayURL:     relayURL,
		payloadMap:   map[common.Hash]*ExecutionPayloadWithTxRootV1{},
	}, nil
}
