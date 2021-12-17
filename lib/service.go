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
	"github.com/fatih/color"
	"github.com/flashbots/mev-middleware/lib/txroot"
	"github.com/gorilla/rpc"
)

var green = color.New(color.FgGreen).SprintFunc()

// MevService TODO
type MevService struct {
	executionURL string
	relayURL     string
}

// RelayService TODO
type RelayService struct {
	executionURL string
	relayURL     string
	store        Store
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
func (m *MevService) ForkchoiceUpdatedV1(r *http.Request, args *[]interface{}, result *catalyst.ForkChoiceResponse) error {
	executionResp, executionErr := makeRequest(m.executionURL, "engine_forkchoiceUpdatedV1", *args)
	relayResp, relayErr := makeRequest(m.relayURL, "engine_forkchoiceUpdatedV1", *args)

	bestResponse := relayResp
	if relayErr != nil {
		log.Println("ForkchoiceUpdatedV1: error in relay resp: ", relayErr, string(relayResp))
		if executionErr != nil {
			// both clients errored, abort
			log.Println("ForkchoiceUpdatedV1: error in both resp: ", executionResp, string(executionResp))
			return relayErr
		}

		bestResponse = executionResp
	}
	resp, err := parseRPCResponse(bestResponse)
	if err != nil {
		log.Println("ForkchoiceUpdatedV1: error parsing result: ", err)
		return err
	}

	err = json.Unmarshal(resp.Result, result)
	if err != nil {
		log.Println("ForkchoiceUpdatedV1: error unmarshaling result: ", err)
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
		log.Println("ExecutePayloadV1: error in relay resp: ", relayErr, string(relayResp))
		if executionErr != nil {
			// both clients errored, abort
			log.Println("ExecutePayloadV1: error in both resp: ", executionResp, string(executionResp))
			return relayErr
		}

		bestResponse = executionResp
	}
	resp, err := parseRPCResponse(bestResponse)
	if err != nil {
		log.Println("ExecutePayloadV1: error parsing result: ", err)
		return err
	}

	err = json.Unmarshal(resp.Result, result)
	if err != nil {
		log.Println("ExecutePayloadV1: error unmarshaling result: ", err)
		return err
	}

	return nil
}

// ProposeBlindedBlockV1 TODO
func (m *RelayService) ProposeBlindedBlockV1(r *http.Request, args *SignedBlindedBeaconBlock, result *ExecutionPayloadWithTxRootV1) error {
	if args == nil || args.Message == nil {
		fmt.Printf("ProposeBlindedBlockV1: blinded block missing body: %+v\n", args)
		return fmt.Errorf("blinded block missing body")
	}

	var body BlindedBeaconBlockBodyPartial
	err := json.Unmarshal(args.Message.Body, &body)
	if err != nil {
		fmt.Printf("ProposeBlindedBlockV1: error parsing body: %+v\n", string(args.Message.Body))
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
		log.Println(green("ProposeBlindedBlockV1: ✓ revealing previous payload from execution client: "), payloadCached.BlockHash, payloadCached.Number, payloadCached.TransactionsRoot)
		*result = *payloadCached
		return nil
	}
	relayResp, relayErr := makeRequest(m.relayURL, "builder_proposeBlindedBlockV1", []interface{}{args})
	if relayErr != nil {
		log.Println("ProposeBlindedBlockV1: error fetching block from relay: ", err)
		return relayErr
	}

	resp, err := parseRPCResponse(relayResp)
	if err != nil {
		log.Println("ProposeBlindedBlockV1: error parsing result: ", err)
		return err
	}

	err = json.Unmarshal(resp.Result, result)
	if err != nil {
		log.Println("ProposeBlindedBlockV1: error unmarshaling result: ", err)
		return err
	}

	log.Println(green("ProposeBlindedBlockV1: ✓ revealing payload from relay: "), result.BlockHash, result.Number)
	return nil
}

var nilHash = common.Hash{}

// GetPayloadHeaderV1 TODO
func (m *RelayService) GetPayloadHeaderV1(r *http.Request, args *string, result *ExecutionPayloadWithTxRootV1) error {
	executionResp, executionErr := makeRequest(m.executionURL, "engine_getPayloadV1", []interface{}{*args})
	relayResp, relayErr := makeRequest(m.relayURL, "engine_getPayloadV1", []interface{}{*args})

	bestResponse := relayResp
	if relayErr != nil {
		log.Println("GetPayloadHeaderV1: error in relay resp: ", relayErr, string(relayResp))
		if executionErr != nil {
			// both clients errored, abort
			log.Println("GetPayloadHeaderV1: error in both resp: ", executionResp, string(executionResp))
			return relayErr
		}

		bestResponse = executionResp
	}

	resp, err := parseRPCResponse(bestResponse)
	if err != nil {
		log.Println("GetPayloadHeaderV1: error parsing result: ", err)
		return err
	}

	err = json.Unmarshal(resp.Result, result)
	if err != nil {
		resp, err = parseRPCResponse(executionResp)
		if err != nil {
			log.Println("GetPayloadHeaderV1: error parsing result: ", err)
			return err
		}

		err = json.Unmarshal(resp.Result, result)
		if err != nil {
			log.Println("GetPayloadHeaderV1: error unmarshaling result: ", err)
			return err
		}
	}

	if result.Transactions != nil {
		log.Println("GetPayloadHeaderV1: no TransactionsRoot found, calculating it from Transactions list instead: ", *args, result.BlockHash, result.Number)

		txs := types.Transactions{}
		byteTxs := [][]byte{}
		for i, otx := range *result.Transactions {
			var tx types.Transaction
			bytesTx := common.Hex2Bytes(otx)
			if err := tx.UnmarshalBinary(bytesTx); err != nil {
				log.Println("GetPayloadHeaderV1: error decoding tx: ", err)
				return fmt.Errorf("failed to decode tx %d: %v", i, err)
			}
			txs = append(txs, &tx)
			byteTxs = append(byteTxs, bytesTx)
		}

		newRootBytes, err := txroot.TransactionsRoot(byteTxs)
		if err != nil {
			log.Println("GetPayloadHeaderV1: error calculating transactions root: ", err)
			return err
		}
		newRoot := common.BytesToHash(newRootBytes[:])

		if result.TransactionsRoot != nilHash {
			if newRoot != result.TransactionsRoot {
				log.Println("GetPayloadHeaderV1: mismatched tx root: ", newRoot.String(), result.TransactionsRoot.String())
				return fmt.Errorf("calculated different transactionsRoot %s: %s", newRoot.String(), result.TransactionsRoot.String())
			}
		}
		result.TransactionsRoot = newRoot

		// copy this payload for later retrieval in proposeBlindedBlock
		payload := new(ExecutionPayloadWithTxRootV1)
		*payload = *result
		m.store.Set(result.BlockHash, payload)
	}
	result.Transactions = nil

	log.Println(green("GetPayloadHeaderV1: ✓ got payload header successfully: "), *args, result.BlockHash, result.Number)
	return nil
}

func (m *MevService) methodNotFound(i *rpc.RequestInfo, w http.ResponseWriter) error {
	log.Println("method not found, forwarding to execution client: ", i.Method)

	req, err := http.NewRequest(http.MethodPost, m.executionURL, bytes.NewReader(i.Body))
	if err != nil {
		log.Println("error in method not found: creating request: ", i.Method, err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("error in method not found: doing request: ", i.Method, err)
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

func newRelayService(executionURL string, relayURL string, store Store) (*RelayService, error) {
	if executionURL == "" {
		return nil, errors.New("NewRouter must have an executionURL")
	}
	if relayURL == "" {
		return nil, errors.New("NewRouter must have an relayURL")
	}

	return &RelayService{
		executionURL: executionURL,
		relayURL:     relayURL,
		store:        store,
	}, nil
}
