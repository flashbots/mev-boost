package server

import (
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethRpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/flashbots/go-utils/jsonrpc"
	"github.com/flashbots/mev-boost/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestBoostRPCServer(relayURLs []string) (*gethRpc.Server, error) {
	return newTestBoostRPCServerWithTimeout(relayURLs, 0)
}

func newTestBoostRPCServerWithTimeout(relayURLs []string, getHeaderTimeout time.Duration) (*gethRpc.Server, error) {
	log := logrus.WithField("testing", true)
	boost, err := NewBoostService(relayURLs, log, getHeaderTimeout)
	if err != nil {
		return nil, err
	}

	srv, err := NewRPCServer("builder", boost)
	if err != nil {
		return nil, err
	}

	return srv, nil
}

func setupMockRelay() *jsonrpc.MockJSONRPCServer {
	server := jsonrpc.NewMockJSONRPCServer()
	server.SetHandler("builder_registerValidatorV1", defaultRegisterValidator)
	return server
}

func defaultRegisterValidator(req *jsonrpc.JSONRPCRequest) (any, error) {
	if len(req.Params) != 2 {
		return nil, fmt.Errorf("Expected 2 params, got %d", len(req.Params))
	}
	return ServiceStatusOk, nil
}

func _hexToBytes(hex string) []byte {
	bytes, err := hexutil.Decode(hex)
	if err != nil {
		panic(err)
	}
	return bytes
}

func TestE2E_RegisterValidator(t *testing.T) {
	relay1, relay2 := setupMockRelay(), setupMockRelay()
	server, err := newTestBoostRPCServer([]string{relay1.URL, relay2.URL})
	require.NoError(t, err)
	defer server.Stop()

	client := gethRpc.DialInProc(server)
	defer client.Close()

	res := ""
	message := types.RegisterValidatorRequestMessage{
		FeeRecipient: common.HexToAddress("0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941"),
		Timestamp:    1234356,
		Pubkey:       _hexToBytes("0xf9716c94aab536227804e859d15207aa7eaaacd839f39dcbdb5adc942842a8d2fb730f9f49fc719fdb86f1873e0ed1c2"),
	}
	err = client.Call(&res, "builder_registerValidatorV1",
		message,
		"0x8682789b16da95ba437a5b51c14ba4e112b50ceacd9730f697c4839b91405280e603fc4367283aa0866af81a21c536c4c452ace2f4146267c5cf6e959955964f4c35f0cedaf80ed99ffc32fe2d28f9390bb30269044fcf20e2dd734c7b287d14", // signature
	)
	require.NoError(t, err)
	require.Equal(t, ServiceStatusOk, res)
	assert.Equal(t, relay1.GetRequestCount("builder_registerValidatorV1"), 1)
	assert.Equal(t, relay2.GetRequestCount("builder_registerValidatorV1"), 1)

	// ---
	// Test one relay returning true, one false (expect true from mev-boost)
	// ---
	relay1.SetHandler("builder_registerValidatorV1", func(req *jsonrpc.JSONRPCRequest) (any, error) { return false, nil })
	err = client.Call(&res, "builder_registerValidatorV1",
		message,
		"0x8682789b16da95ba437a5b51c14ba4e112b50ceacd9730f697c4839b91405280e603fc4367283aa0866af81a21c536c4c452ace2f4146267c5cf6e959955964f4c35f0cedaf80ed99ffc32fe2d28f9390bb30269044fcf20e2dd734c7b287d14", // signature
	)
	require.NoError(t, err)
	require.Equal(t, ServiceStatusOk, res)
	assert.Equal(t, relay1.GetRequestCount("builder_registerValidatorV1"), 2)
	assert.Equal(t, relay2.GetRequestCount("builder_registerValidatorV1"), 2)

	// ---
	// Test both relays returning false (expect false from mev-boost)
	// ---
	relay2.SetHandler("builder_registerValidatorV1", func(req *jsonrpc.JSONRPCRequest) (any, error) { return false, nil })
	err = client.Call(&res, "builder_registerValidatorV1",
		message,
		"0x8682789b16da95ba437a5b51c14ba4e112b50ceacd9730f697c4839b91405280e603fc4367283aa0866af81a21c536c4c452ace2f4146267c5cf6e959955964f4c35f0cedaf80ed99ffc32fe2d28f9390bb30269044fcf20e2dd734c7b287d14", // signature
	)
	require.NotNil(t, err, err)
	assert.Equal(t, relay1.GetRequestCount("builder_registerValidatorV1"), 3)
	assert.Equal(t, relay2.GetRequestCount("builder_registerValidatorV1"), 3)
}

// Ensure mev-boost catches an invalid payload (invalid number of params)
func TestE2E_RegisterValidator_Error(t *testing.T) {
	relay1 := setupMockRelay()
	server, err := newTestBoostRPCServer([]string{relay1.URL})
	require.NoError(t, err)
	defer server.Stop()

	client := gethRpc.DialInProc(server)
	defer client.Close()

	// Invalid number of arguments
	res := ""
	err = client.Call(&res, "builder_registerValidatorV1",
		"0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941", // feeRecipient
		"0x625481c2", // timestamp
		"0x01",
	)
	require.NotNil(t, err, err)
	assert.Equal(t, relay1.GetRequestCount("builder_registerValidatorV1"), 0)

	// Test invalid message type
	err = client.Call(&res, "builder_registerValidatorV1",
		"0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941", // feeRecipient
		"0x625481c2", // timestamp
		"0x01",
	)
	require.Error(t, err)
	ec, isErrorWithCode := err.(errorWithCode)
	require.True(t, isErrorWithCode)
	require.Equal(t, -32602, ec.ErrorCode())
	assert.Equal(t, relay1.GetRequestCount("builder_registerValidatorV1"), 0)

	// Invalid message type
	err = client.Call(&res, "builder_registerValidatorV1",
		"0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941", // feeRecipient
		"0x625481c2", // timestamp
	)
	require.Error(t, err)
	ec, isErrorWithCode = err.(errorWithCode)
	require.True(t, isErrorWithCode)
	require.Equal(t, -32602, ec.ErrorCode())
	assert.Equal(t, relay1.GetRequestCount("builder_registerValidatorV1"), 0)

	// Invalid signature
	err = client.Call(&res, "builder_registerValidatorV1",
		types.RegisterValidatorRequestMessage{
			FeeRecipient: common.HexToAddress("0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941"),
			Timestamp:    1234356,
			Pubkey:       _hexToBytes("0xf9716c94aab536227804e859d15207aa7eaaacd839f39dcbdb5adc942842a8d2fb730f9f49fc719fdb86f1873e0ed1c2"),
		},
		"0x8682789b16da95ba437a5b51c14ba4e112b50ceacd9730f697c4839b91405280e603fc4367283aa0866af81a21c536c4c452ace2f4146267c5cf6e959955964f4c35f0cedaf80ed99ffc32fe2d28f9390bb30269044fcf20e2dd734c7b287d",
	)
	require.Error(t, err)
	ec, isErrorWithCode = err.(errorWithCode)
	require.True(t, isErrorWithCode)
	require.Equal(t, -32602, ec.ErrorCode())
	assert.Equal(t, relay1.GetRequestCount("builder_registerValidatorV1"), 0)

	// no relay returned ok
	relay1.SetHandler("builder_registerValidatorV1", func(req *jsonrpc.JSONRPCRequest) (interface{}, error) {
		return nil, errors.New("test error")
	})
	err = client.Call(&res, "builder_registerValidatorV1",
		types.RegisterValidatorRequestMessage{
			FeeRecipient: common.HexToAddress("0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941"),
			Timestamp:    1234356,
			Pubkey:       _hexToBytes("0xf9716c94aab536227804e859d15207aa7eaaacd839f39dcbdb5adc942842a8d2fb730f9f49fc719fdb86f1873e0ed1c2"),
		},
		"0x8682789b16da95ba437a5b51c14ba4e112b50ceacd9730f697c4839b91405280e603fc4367283aa0866af81a21c536c4c452ace2f4146267c5cf6e959955964f4c35f0cedaf80ed99ffc32fe2d28f9390bb30269044fcf20e2dd734c7b287d14",
	)
	require.Error(t, err)
	ec, isErrorWithCode = err.(errorWithCode)
	require.True(t, isErrorWithCode)
	require.Equal(t, -32603, ec.ErrorCode())
	assert.Equal(t, relay1.GetRequestCount("builder_registerValidatorV1"), 1)
}

// Ensure mev-boost passes on error codes from relay
func Test_RelayErrorCodePassthrough(t *testing.T) {
	relay1 := setupMockRelay()
	relay1.SetHandler("builder_registerValidatorV1", func(req *jsonrpc.JSONRPCRequest) (interface{}, error) {
		return nil, rpcError{Code: -123, Message: "test error"}
	})
	server, err := newTestBoostRPCServer([]string{relay1.URL})

	require.NoError(t, err)
	defer server.Stop()

	client := gethRpc.DialInProc(server)
	defer client.Close()

	// Invalid number of arguments
	res := ""
	err = client.Call(&res, "builder_registerValidatorV1",
		types.RegisterValidatorRequestMessage{
			FeeRecipient: common.HexToAddress("0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941"),
			Timestamp:    1234356,
			Pubkey:       _hexToBytes("0xf9716c94aab536227804e859d15207aa7eaaacd839f39dcbdb5adc942842a8d2fb730f9f49fc719fdb86f1873e0ed1c2"),
		},
		"0x8682789b16da95ba437a5b51c14ba4e112b50ceacd9730f697c4839b91405280e603fc4367283aa0866af81a21c536c4c452ace2f4146267c5cf6e959955964f4c35f0cedaf80ed99ffc32fe2d28f9390bb30269044fcf20e2dd734c7b287d14", // signature
	)

	// Must return an error, and then ensure it also has a code that matches the relay error code
	require.Error(t, err, err)
	ec, isErrorWithCode := err.(errorWithCode)
	require.True(t, isErrorWithCode)
	require.Equal(t, -123, ec.ErrorCode())

	assert.Equal(t, relay1.GetRequestCount("builder_registerValidatorV1"), 1)
}

// Ensure that mev-boost forwards the last error response from the relay if all relays return an error
func TestE2E_RegisterValidator_RelayError(t *testing.T) {
	relay1, relay2 := setupMockRelay(), setupMockRelay()
	testErr := &jsonrpc.JSONRPCError{Code: -32009, Message: "test error"}
	relay1.SetHandler("builder_registerValidatorV1", func(req *jsonrpc.JSONRPCRequest) (interface{}, error) {
		return nil, testErr
	})
	relay2.SetHandler("builder_registerValidatorV1", func(req *jsonrpc.JSONRPCRequest) (interface{}, error) {
		return nil, testErr
	})

	server, err := newTestBoostRPCServer([]string{relay1.URL, relay2.URL})
	require.NoError(t, err)
	defer server.Stop()

	client := gethRpc.DialInProc(server)
	defer client.Close()

	res := false
	err = client.Call(&res, "builder_registerValidatorV1",
		types.RegisterValidatorRequestMessage{
			FeeRecipient: common.HexToAddress("0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941"),
			Timestamp:    1234567,
			Pubkey:       _hexToBytes("0xf9716c94aab536227804e859d15207aa7eaaacd839f39dcbdb5adc942842a8d2fb730f9f49fc719fdb86f1873e0ed1c2"),
		},
		"0xab5dc3c47ea96503823f364c4c1bb747560dc8874d90acdd0cbcfe1abc5457a70ab7e8175c074ace44dead2427e6d2353184c61c6eebc3620b8cec1e9115e35e4513369d7a68d7a5dad719cb6f5a85788490f76ca3580758042da4d003ef373f", // signature
	)
	require.NotNil(t, err, err)
	assert.Equal(t, relay1.GetRequestCount("builder_registerValidatorV1"), 1)
	assert.Equal(t, relay2.GetRequestCount("builder_registerValidatorV1"), 1)
}

// builder for a getHeader handler with a custom value
func makeBuilderGetHeaderV1Handler(value int64, parentHash common.Hash, delay time.Duration) func(req *jsonrpc.JSONRPCRequest) (any, error) {
	_value := big.NewInt(value)
	return func(req *jsonrpc.JSONRPCRequest) (any, error) {
		if delay > 0 {
			time.Sleep(delay)
		}
		if len(req.Params) != 3 {
			return nil, fmt.Errorf("Expected 3 params, got %d", len(req.Params))
		}
		resp := &types.GetHeaderResponse{
			Message: types.GetHeaderResponseMessage{
				Header: types.ExecutionPayloadHeaderV1{
					ParentHash:    parentHash,
					BlockHash:     common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
					BaseFeePerGas: big.NewInt(4),
				},
				Value:  (*hexutil.Big)(_value),
				Pubkey: _hexToBytes("0x1bf4b68731b493e9474f0cd5d0faaeed8796ac77267d93949c96cff6dcfaf04e49f32a7ebce3ba132b58b6f17ec5754a"),
			},
			Signature: _hexToBytes("0x258c63ecdc711c29f11174ebc67bcbb46efa2bda3f1a6315a28bb74b1e6d8a442778c4880f1c2e9cdaca8e3a1370a9e203fcbf45660edc61011ae9df66912c133fdaa15a917041bc7807326b42db8e6c052a7e9cb5ca9c17181952837809bb51"),
		}
		return resp, nil
	}
}

func TestE2E_GetHeader(t *testing.T) {
	relay1, relay2 := setupMockRelay(), setupMockRelay()
	server, err := newTestBoostRPCServer([]string{relay1.URL, relay2.URL})
	require.NoError(t, err)
	defer server.Stop()

	client := gethRpc.DialInProc(server)
	defer client.Close()

	parentHash := common.HexToHash("0xf254722f498df7e396694ed71f363c535ae1b2620afeaf57515e7593ad888331")

	// Set handlers with different values
	relay1.SetHandler("builder_getHeaderV1", makeBuilderGetHeaderV1Handler(12345, parentHash, 0))
	relay2.SetHandler("builder_getHeaderV1", makeBuilderGetHeaderV1Handler(12345678, parentHash, 0))

	res := new(types.GetHeaderResponse)
	pubkey := "0xf0d89fec26f5e1e84884a293677ee7e7d48505a43d23f7e4888206a780fe33ccaf374b317eb78c7036cb6c97af1dfe9a"
	err = client.Call(&res, "builder_getHeaderV1", "0x1", pubkey, parentHash)
	require.NoError(t, err)
	require.Equal(t, parentHash, res.Message.Header.ParentHash)
	assert.Equal(t, relay1.GetRequestCount("builder_getHeaderV1"), 1)
	assert.Equal(t, relay2.GetRequestCount("builder_getHeaderV1"), 1)
	assert.Equal(t, "12345678", res.Message.Value.ToInt().String())

	// ---
	// Test with a slow relay - ensuring that a specific GetHeaderTimeout is respected.
	// relay2 responds with a delay longer than GetHeaderTimeout. Therefore only response from relay1 is used.
	// ---
	server2, err := newTestBoostRPCServerWithTimeout([]string{relay1.URL, relay2.URL}, time.Millisecond*100)
	require.NoError(t, err)

	client2 := gethRpc.DialInProc(server2)
	defer client2.Close()

	relay2.SetHandler("builder_getHeaderV1", makeBuilderGetHeaderV1Handler(12345678, parentHash, 110*time.Millisecond))
	err = client2.Call(&res, "builder_getHeaderV1", "0x1", pubkey, parentHash.Hex())
	require.NoError(t, err)
	assert.Equal(t, relay1.GetRequestCount("builder_getHeaderV1"), 2)
	assert.Equal(t, relay2.GetRequestCount("builder_getHeaderV1"), 2)
	assert.Equal(t, "12345", res.Message.Value.ToInt().String())
}

func TestE2E_GetHeaderError(t *testing.T) {
	parentHash := common.HexToHash("0xf254722f498df7e396694ed71f363c535ae1b2620afeaf57515e7593ad888331")
	relay1 := setupMockRelay()
	relay1.SetHandler("builder_getHeaderV1", makeBuilderGetHeaderV1Handler(12345, parentHash, 0))

	server, err := newTestBoostRPCServer([]string{relay1.URL})
	require.NoError(t, err)
	defer server.Stop()

	client := gethRpc.DialInProc(server)
	defer client.Close()

	res := new(types.GetHeaderResponse)
	err = client.Call(&res, "builder_getHeaderV1", "0x1", "0x1bf4b68731b493e9474f0cd5d0faaeed8796ac77267d93949c96cff6dcfaf04e49f32a7ebce3ba132b58b6f17ec575", "0xf254722f498df7e396694ed71f363c535ae1b2620afeaf57515e7593ad888331")
	require.Error(t, err)
	require.Equal(t, rpcErrInvalidPubkey.Error(), err.Error())
}

func TestE2E_GetPayload(t *testing.T) {
	relay1, relay2 := setupMockRelay(), setupMockRelay()
	server, err := newTestBoostRPCServer([]string{relay1.URL, relay2.URL})
	require.NoError(t, err)
	defer server.Stop()

	client := gethRpc.DialInProc(server)
	defer client.Close()

	// builder for a getHeader handler with a custom value
	getPayloadV1Handler := func(req *jsonrpc.JSONRPCRequest) (any, error) {
		resp := &types.ExecutionPayloadV1{
			BlockHash:     common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
			BaseFeePerGas: big.NewInt(4),
			Transactions:  &[]string{},
		}
		return resp, nil
	}

	// Set handlers with different values
	relay1.SetHandler("builder_getPayloadV1", getPayloadV1Handler)
	relay2.SetHandler("builder_getPayloadV1", getPayloadV1Handler)

	res := new(types.ExecutionPayloadV1)
	err = client.Call(&res, "builder_getPayloadV1",
		&types.BlindedBeaconBlockV1{
			Slot: "0x1",
			Body: types.BlindedBeaconBlockBodyV1{
				ExecutionPayloadHeader: types.ExecutionPayloadHeaderV1{
					BaseFeePerGas: big.NewInt(4),
				},
			},
		},
		"0x8682789b16da95ba437a5b51c14ba4e112b50ceacd9730f697c4839b91405280e603fc4367283aa0866af81a21c536c4c452ace2f4146267c5cf6e959955964f4c35f0cedaf80ed99ffc32fe2d28f9390bb30269044fcf20e2dd734c7b287d14",
	)
	require.NoError(t, err)
	assert.Equal(t, relay1.GetRequestCount("builder_getPayloadV1"), 1)
	assert.Equal(t, relay2.GetRequestCount("builder_getPayloadV1"), 1)
}

func TestE2E_Status(t *testing.T) {
	relay1, relay2 := setupMockRelay(), setupMockRelay()
	server, err := newTestBoostRPCServer([]string{relay1.URL, relay2.URL})
	require.NoError(t, err)
	defer server.Stop()

	client := gethRpc.DialInProc(server)
	defer client.Close()

	res := ""
	err = client.Call(&res, "builder_status")
	require.NoError(t, err)
	require.Equal(t, "OK", res)
}
