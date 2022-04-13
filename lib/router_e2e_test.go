package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flashbots/go-utils/jsonrpc"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDefaultRouter(relayURLs []string) (*mux.Router, error) {
	return NewRouter(RouterOptions{
		RelayURLs: relayURLs,
		Log:       logrus.WithField("testing", true),
	})
}

func setupMockRelay() *jsonrpc.MockJSONRPCServer {
	server := jsonrpc.NewMockJSONRPCServer()
	server.SetHandler("builder_setFeeRecipientV1", defaultSetFeeRecipient)
	return server
}

func defaultSetFeeRecipient(req *jsonrpc.JSONRPCRequest) (any, error) {
	if len(req.Params) != 4 {
		return false, fmt.Errorf("Expected 4 params, got %d", len(req.Params))
	}
	return true, nil
}

func sendJSONRPCRequestToRouter(router *mux.Router, req *rpcRequest) (*rpcResponse, error) {
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	_req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(buf))
	_req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Actually send the request, testing the router
	router.ServeHTTP(w, _req)
	// fmt.Println(w.Body.String())
	resp, err := parseRPCResponse(w.Body.Bytes())
	return resp, err
}

func sendRequestFailOnError(t *testing.T, router *mux.Router, req *rpcRequest) *rpcResponse {
	resp, err := sendJSONRPCRequestToRouter(router, req)
	require.Nil(t, err, err)
	require.Nil(t, resp.Error, resp.Error)
	return resp
}

func TestE2E_SetFeeRecipient(t *testing.T) {
	relay1, relay2 := setupMockRelay(), setupMockRelay()
	router, err := newDefaultRouter([]string{relay1.URL, relay2.URL})
	require.Nil(t, err, err)

	req := newRPCRequest("1", "builder_setFeeRecipientV1", []any{
		"0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941", // feeRecipient
		"0x625481c2", // timestamp
		"0xf9716c94aab536227804e859d15207aa7eaaacd839f39dcbdb5adc942842a8d2fb730f9f49fc719fdb86f1873e0ed1c2",                                                                                                 // pubkey
		"0xab5dc3c47ea96503823f364c4c1bb747560dc8874d90acdd0cbcfe1abc5457a70ab7e8175c074ace44dead2427e6d2353184c61c6eebc3620b8cec1e9115e35e4513369d7a68d7a5dad719cb6f5a85788490f76ca3580758042da4d003ef373f", // signature
	})

	resp := sendRequestFailOnError(t, router, req)

	result := false
	err = json.Unmarshal(resp.Result, &result)
	require.Nil(t, err, err)
	require.Equal(t, true, result)

	assert.Equal(t, relay1.RequestCounter["builder_setFeeRecipientV1"], 1)
	assert.Equal(t, relay2.RequestCounter["builder_setFeeRecipientV1"], 1)

	// ---
	// Test one relay returning true, one false (expect true from mev-boost)
	// ---
	relay1.SetHandler("builder_setFeeRecipientV1", func(req *jsonrpc.JSONRPCRequest) (any, error) { return false, nil })
	resp = sendRequestFailOnError(t, router, req)
	err = json.Unmarshal(resp.Result, &result)
	require.Nil(t, err, err)
	require.Equal(t, true, result)
	assert.Equal(t, relay1.RequestCounter["builder_setFeeRecipientV1"], 2)
	assert.Equal(t, relay2.RequestCounter["builder_setFeeRecipientV1"], 2)

	// ---
	// Test both relays returning false (expect false from mev-boost)
	// ---
	relay2.SetHandler("builder_setFeeRecipientV1", func(req *jsonrpc.JSONRPCRequest) (any, error) { return false, nil })
	resp, err = sendJSONRPCRequestToRouter(router, req)
	require.Nil(t, err, err)
	require.NotNil(t, resp.Error)

	assert.Equal(t, relay1.RequestCounter["builder_setFeeRecipientV1"], 3)
	assert.Equal(t, relay2.RequestCounter["builder_setFeeRecipientV1"], 3)
}

// Ensure mev-boost catches an invalid payload (invalid number of params)
func TestE2E_SetFeeRecipient_Error(t *testing.T) {
	relay1 := setupMockRelay()
	router, err := newDefaultRouter([]string{relay1.URL})
	require.Nil(t, err, err)

	req := newRPCRequest("1", "builder_setFeeRecipientV1", []any{
		"0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941", // feeRecipient
		"0x625481c2", // timestamp
	})

	resp, err := sendJSONRPCRequestToRouter(router, req)
	require.Nil(t, err, err)
	require.NotNil(t, resp.Error, resp.Error)
	require.Contains(t, resp.Error.Message, "invalid number of params")
	assert.Equal(t, relay1.RequestCounter["builder_setFeeRecipientV1"], 0)
}

// Ensure that mev-boost forwards the last error response from the relay if all relays return an error
func TestE2E_SetFeeRecipient_RelayError(t *testing.T) {
	relay1, relay2 := setupMockRelay(), setupMockRelay()
	router, err := newDefaultRouter([]string{relay1.URL, relay2.URL})
	require.Nil(t, err, err)

	// Set relay handlers that return an error
	testErr := &jsonrpc.JSONRPCError{Code: -32009, Message: "test error"}
	relay1.SetHandler("builder_setFeeRecipientV1", func(req *jsonrpc.JSONRPCRequest) (interface{}, error) {
		return nil, testErr
	})
	relay2.SetHandler("builder_setFeeRecipientV1", func(req *jsonrpc.JSONRPCRequest) (interface{}, error) {
		return nil, testErr
	})

	req := newRPCRequest("1", "builder_setFeeRecipientV1", []any{
		"0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941", // feeRecipient
		"0x625481c2", // timestamp
		"0xf9716c94aab536227804e859d15207aa7eaaacd839f39dcbdb5adc942842a8d2fb730f9f49fc719fdb86f1873e0ed1c2",                                                                                                 // pubkey
		"0xab5dc3c47ea96503823f364c4c1bb747560dc8874d90acdd0cbcfe1abc5457a70ab7e8175c074ace44dead2427e6d2353184c61c6eebc3620b8cec1e9115e35e4513369d7a68d7a5dad719cb6f5a85788490f76ca3580758042da4d003ef373f", // signature
	})

	resp, err := sendJSONRPCRequestToRouter(router, req)
	require.Nil(t, err, err)
	require.NotNil(t, resp.Error) // mev-boost will have returned the latest error
	// require.Equal(t, -32009, resp.Error.Code) // TODO: swtich to go-ethereum/rpc
	// require.Equal(t, "relay error", resp.Error.Message) // TODO: swtich to go-ethereum/rpc
	assert.Equal(t, relay1.RequestCounter["builder_setFeeRecipientV1"], 1)
	assert.Equal(t, relay2.RequestCounter["builder_setFeeRecipientV1"], 1)
}

func TestE2E_GetHeader(t *testing.T) {
	relay1, relay2 := setupMockRelay(), setupMockRelay()
	router, err := newDefaultRouter([]string{relay1.URL, relay2.URL})
	require.Nil(t, err, err)

	parentHash := common.HexToHash("0xf254722f498df7e396694ed71f363c535ae1b2620afeaf57515e7593ad888331")

	// builder for a getHeader handler with a custom value
	makeBuilderGetHeaderV1Handler := func(value *big.Int, delay time.Duration) func(req *jsonrpc.JSONRPCRequest) (any, error) {
		return func(req *jsonrpc.JSONRPCRequest) (any, error) {
			if delay > 0 {
				time.Sleep(delay)
			}
			if len(req.Params) != 1 {
				return nil, fmt.Errorf("Expected 1 params, got %d", len(req.Params))
			}
			assert.Equal(t, parentHash.String(), req.Params[0].(string))
			resp := &GetHeaderResponse{
				Header: ExecutionPayloadHeaderV1{
					ParentHash:    parentHash,
					BlockHash:     common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
					BaseFeePerGas: big.NewInt(4),
				},
				Value:     value,
				PublicKey: []byte{0x1},
				Signature: []byte{0x2},
			}
			return resp, nil
		}
	}

	// Set handlers with different values
	relay1.SetHandler("builder_getHeaderV1", makeBuilderGetHeaderV1Handler(big.NewInt(12345), 0))
	relay2.SetHandler("builder_getHeaderV1", makeBuilderGetHeaderV1Handler(big.NewInt(12345678), 0))

	req := newRPCRequest("1", "builder_getHeaderV1", []any{parentHash.Hex()})
	resp := sendRequestFailOnError(t, router, req)
	assert.Equal(t, relay1.RequestCounter["builder_getHeaderV1"], 1)
	assert.Equal(t, relay2.RequestCounter["builder_getHeaderV1"], 1)

	result := new(GetHeaderResponse)
	err = json.Unmarshal(resp.Result, result)
	require.Nil(t, err, err)
	assert.Equal(t, "12345678", result.Value.String())

	// ---
	// Test with a slow relay - ensuring that a specific GetHeaderTimeout is respected.
	// relay2 responds with a delay longer than GetHeaderTimeout. Therefore only response from relay1 is used.
	// ---
	router, err = NewRouter(RouterOptions{
		RelayURLs:        []string{relay1.URL, relay2.URL},
		Log:              logrus.WithField("testing", true),
		GetHeaderTimeout: 100 * time.Millisecond,
	})
	require.Nil(t, err, err)

	relay2.SetHandler("builder_getHeaderV1", makeBuilderGetHeaderV1Handler(big.NewInt(12345678), 110*time.Millisecond))
	req = newRPCRequest("1", "builder_getHeaderV1", []any{parentHash})
	resp = sendRequestFailOnError(t, router, req)
	assert.Equal(t, relay1.RequestCounter["builder_getHeaderV1"], 2)
	assert.Equal(t, relay2.RequestCounter["builder_getHeaderV1"], 2)

	result = new(GetHeaderResponse)
	err = json.Unmarshal(resp.Result, result)
	require.Nil(t, err, err)
	assert.Equal(t, "12345", result.Value.String())
}

func TestE2E_GetPayload(t *testing.T) {
	relay1, relay2 := setupMockRelay(), setupMockRelay()
	router, err := newDefaultRouter([]string{relay1.URL, relay2.URL})
	require.Nil(t, err, err)

	// builder for a getHeader handler with a custom value
	getPayloadV1Handler := func(req *jsonrpc.JSONRPCRequest) (any, error) {
		resp := &ExecutionPayloadV1{
			BlockHash:     common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
			BaseFeePerGas: big.NewInt(4),
			Transactions:  &[]string{},
		}
		return resp, nil
	}

	// Set handlers with different values
	relay1.SetHandler("builder_getPayloadV1", getPayloadV1Handler)
	relay2.SetHandler("builder_getPayloadV1", getPayloadV1Handler)

	req := newRPCRequest("1", "builder_getPayloadV1", []any{"0x0000000000000000000000000000000000000000000000000000000000000001", "0x0000000000000000000000000000000000000000000000000000000000000002"})
	resp := sendRequestFailOnError(t, router, req)
	assert.Equal(t, relay1.RequestCounter["builder_getPayloadV1"], 1)
	assert.Equal(t, relay2.RequestCounter["builder_getPayloadV1"], 1)

	result := new(ExecutionPayloadV1)
	err = json.Unmarshal(resp.Result, result)
	require.Nil(t, err, err)
}
