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

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/gorilla/rpc"
)

// MevService TODO
type MevService struct {
	executionURL string
	relayURL     string
}

// RelayService TODO
type RelayService struct {
	relayURL string
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

// GetPayloadV1 TODO
// TODO: PayloadID is changed to hexutil.Bytes in upstream?
func (m *MevService) GetPayloadV1(r *http.Request, args *hexutil.Uint64, result *catalyst.ExecutableDataV1) error {
	executionResp, executionErr := makeRequest(m.executionURL, "engine_getPayloadV1", []interface{}{args.String()})
	relayResp, relayErr := makeRequest(m.relayURL, "engine_getPayloadV1", []interface{}{args.String()})

	bestResponse := relayResp
	if relayErr != nil {
		log.Print("error in relay resp", relayErr, string(relayResp))
		if executionErr != nil {
			// both clients errored, abort
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
		fmt.Println("error unmarshaling result", err)
		return err
	}

	return nil
}

// ForkchoiceUpdatedV1 TODO
func (m *MevService) ForkchoiceUpdatedV1(r *http.Request, args *catalyst.PayloadAttributesV1, result *catalyst.ForkChoiceResponse) error {
	executionResp, executionErr := makeRequest(m.executionURL, "engine_forkchoiceUpdatedV1", []interface{}{args})
	relayResp, relayErr := makeRequest(m.relayURL, "engine_forkchoiceUpdatedV1", []interface{}{args})

	bestResponse := relayResp
	if relayErr != nil {
		log.Print("error in relay resp", relayErr, string(relayResp))
		if executionErr != nil {
			// both clients errored, abort
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
		fmt.Println("error unmarshaling result", err)
		return err
	}

	return nil
}

// ExecutePayloadV1 TODO
func (m *MevService) ExecutePayloadV1(r *http.Request, args *catalyst.ExecutableDataV1, result *catalyst.ExecutePayloadResponse) error {
	executionResp, executionErr := makeRequest(m.executionURL, "engine_executePayloadV1", []interface{}{args})
	relayResp, relayErr := makeRequest(m.relayURL, "engine_executePayloadV1", []interface{}{args})

	bestResponse := relayResp
	if relayErr != nil {
		log.Print("error in relay resp", relayErr, string(relayResp))
		if executionErr != nil {
			// both clients errored, abort
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
		fmt.Println("error unmarshaling result", err)
		return err
	}

	return nil
}

// ProposePayloadV1 TODO
func (m *RelayService) ProposePayloadV1(r *http.Request, args *catalyst.ExecutableDataV1, result *catalyst.ExecutePayloadResponse) error {
	relayResp, relayErr := makeRequest(m.relayURL, "relay_proposePayloadV1", []interface{}{args})
	if relayErr != nil {
		return relayErr
	}

	resp, err := parseRPCResponse(relayResp)
	if err != nil {
		return err
	}

	err = json.Unmarshal(resp.Result, result)
	if err != nil {
		fmt.Println("error unmarshaling result", err)
		return err
	}

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

func newRelayService(relayURL string) (*RelayService, error) {

	if relayURL == "" {
		return nil, errors.New("NewRouter must have an relayURL")
	}

	return &RelayService{
		relayURL: relayURL,
	}, nil
}
