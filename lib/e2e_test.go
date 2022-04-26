package lib

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
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
	boost, err := newBoostService(relayURLs, log, getHeaderTimeout)
	if err != nil {
		return nil, err
	}

	srv, err := NewRPCServer("builder", boost, true)
	if err != nil {
		return nil, err
	}

	return srv, nil
}

func setupMockRelay() *jsonrpc.MockJSONRPCServer {
	server := jsonrpc.NewMockJSONRPCServer()
	server.SetHandler("builder_setFeeRecipientV1", defaultSetFeeRecipient)
	return server
}

func defaultSetFeeRecipient(req *jsonrpc.JSONRPCRequest) (any, error) {
	if len(req.Params) != 3 {
		return false, fmt.Errorf("Expected 3 params, got %d", len(req.Params))
	}
	return true, nil
}

func TestE2E_SetFeeRecipient(t *testing.T) {
	relay1, relay2 := setupMockRelay(), setupMockRelay()
	server, err := newTestBoostRPCServer([]string{relay1.URL, relay2.URL})
	require.Nil(t, err, err)
	defer server.Stop()

	client := gethRpc.DialInProc(server)
	defer client.Close()

	res := false
	message := types.SetFeeRecipientMessage{
		FeeRecipient: "0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941",
		Timestamp:    "0x625481c2",
	}
	err = client.Call(&res, "builder_setFeeRecipientV1",
		message,
		"0xf9716c94aab536227804e859d15207aa7eaaacd839f39dcbdb5adc942842a8d2fb730f9f49fc719fdb86f1873e0ed1c2",                                                                                                 // pubkey
		"0xab5dc3c47ea96503823f364c4c1bb747560dc8874d90acdd0cbcfe1abc5457a70ab7e8175c074ace44dead2427e6d2353184c61c6eebc3620b8cec1e9115e35e4513369d7a68d7a5dad719cb6f5a85788490f76ca3580758042da4d003ef373f", // signature
	)
	require.Nil(t, err, err)
	require.True(t, res)
	assert.Equal(t, relay1.RequestCounter["builder_setFeeRecipientV1"], 1)
	assert.Equal(t, relay2.RequestCounter["builder_setFeeRecipientV1"], 1)

	// ---
	// Test one relay returning true, one false (expect true from mev-boost)
	// ---
	relay1.SetHandler("builder_setFeeRecipientV1", func(req *jsonrpc.JSONRPCRequest) (any, error) { return false, nil })
	err = client.Call(&res, "builder_setFeeRecipientV1",
		message,
		"0xf9716c94aab536227804e859d15207aa7eaaacd839f39dcbdb5adc942842a8d2fb730f9f49fc719fdb86f1873e0ed1c2",                                                                                                 // pubkey
		"0xab5dc3c47ea96503823f364c4c1bb747560dc8874d90acdd0cbcfe1abc5457a70ab7e8175c074ace44dead2427e6d2353184c61c6eebc3620b8cec1e9115e35e4513369d7a68d7a5dad719cb6f5a85788490f76ca3580758042da4d003ef373f", // signature
	)
	require.Nil(t, err, err)
	require.True(t, res)
	assert.Equal(t, relay1.RequestCounter["builder_setFeeRecipientV1"], 2)
	assert.Equal(t, relay2.RequestCounter["builder_setFeeRecipientV1"], 2)

	// ---
	// Test both relays returning false (expect false from mev-boost)
	// ---
	relay2.SetHandler("builder_setFeeRecipientV1", func(req *jsonrpc.JSONRPCRequest) (any, error) { return false, nil })
	err = client.Call(&res, "builder_setFeeRecipientV1",
		message,
		"0xf9716c94aab536227804e859d15207aa7eaaacd839f39dcbdb5adc942842a8d2fb730f9f49fc719fdb86f1873e0ed1c2",                                                                                                 // pubkey
		"0xab5dc3c47ea96503823f364c4c1bb747560dc8874d90acdd0cbcfe1abc5457a70ab7e8175c074ace44dead2427e6d2353184c61c6eebc3620b8cec1e9115e35e4513369d7a68d7a5dad719cb6f5a85788490f76ca3580758042da4d003ef373f", // signature
	)
	require.NotNil(t, err, err)
	assert.Equal(t, relay1.RequestCounter["builder_setFeeRecipientV1"], 3)
	assert.Equal(t, relay2.RequestCounter["builder_setFeeRecipientV1"], 3)
}

// Ensure mev-boost catches an invalid payload (invalid number of params)
func TestE2E_SetFeeRecipient_Error(t *testing.T) {
	relay1 := setupMockRelay()
	server, err := newTestBoostRPCServer([]string{relay1.URL})
	require.Nil(t, err, err)
	defer server.Stop()

	client := gethRpc.DialInProc(server)
	defer client.Close()

	res := false
	err = client.Call(&res, "builder_setFeeRecipientV1",
		"0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941", // feeRecipient
		"0x625481c2", // timestamp
	)
	require.NotNil(t, err, err)
	assert.Equal(t, relay1.RequestCounter["builder_setFeeRecipientV1"], 0)
}

// Ensure that mev-boost forwards the last error response from the relay if all relays return an error
func TestE2E_SetFeeRecipient_RelayError(t *testing.T) {
	relay1, relay2 := setupMockRelay(), setupMockRelay()
	testErr := &jsonrpc.JSONRPCError{Code: -32009, Message: "test error"}
	relay1.SetHandler("builder_setFeeRecipientV1", func(req *jsonrpc.JSONRPCRequest) (interface{}, error) {
		return nil, testErr
	})
	relay2.SetHandler("builder_setFeeRecipientV1", func(req *jsonrpc.JSONRPCRequest) (interface{}, error) {
		return nil, testErr
	})

	server, err := newTestBoostRPCServer([]string{relay1.URL, relay2.URL})
	require.Nil(t, err, err)
	defer server.Stop()

	client := gethRpc.DialInProc(server)
	defer client.Close()

	res := false
	err = client.Call(&res, "builder_setFeeRecipientV1",
		types.SetFeeRecipientMessage{
			FeeRecipient: "0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941",
			Timestamp:    "0x123",
		},
		"0xf9716c94aab536227804e859d15207aa7eaaacd839f39dcbdb5adc942842a8d2fb730f9f49fc719fdb86f1873e0ed1c2",                                                                                                 // pubkey
		"0xab5dc3c47ea96503823f364c4c1bb747560dc8874d90acdd0cbcfe1abc5457a70ab7e8175c074ace44dead2427e6d2353184c61c6eebc3620b8cec1e9115e35e4513369d7a68d7a5dad719cb6f5a85788490f76ca3580758042da4d003ef373f", // signature
	)
	require.NotNil(t, err, err)
	assert.Equal(t, relay1.RequestCounter["builder_setFeeRecipientV1"], 1)
	assert.Equal(t, relay2.RequestCounter["builder_setFeeRecipientV1"], 1)
}

func TestE2E_GetHeader(t *testing.T) {
	relay1, relay2 := setupMockRelay(), setupMockRelay()
	server, err := newTestBoostRPCServer([]string{relay1.URL, relay2.URL})
	require.Nil(t, err, err)
	defer server.Stop()

	client := gethRpc.DialInProc(server)
	defer client.Close()

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
			resp := &types.GetHeaderResponse{
				Message: types.GetHeaderResponseMessage{
					Header: types.ExecutionPayloadHeaderV1{
						ParentHash:    parentHash,
						BlockHash:     common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
						BaseFeePerGas: big.NewInt(4),
					},
					Value: value,
				},
				PublicKey: []byte{0x1},
				Signature: []byte{0x2},
			}
			return resp, nil
		}
	}

	// Set handlers with different values
	relay1.SetHandler("builder_getHeaderV1", makeBuilderGetHeaderV1Handler(big.NewInt(12345), 0))
	relay2.SetHandler("builder_getHeaderV1", makeBuilderGetHeaderV1Handler(big.NewInt(12345678), 0))

	res := new(types.GetHeaderResponse)
	err = client.Call(&res, "builder_getHeaderV1", parentHash.Hex())
	require.Nil(t, err, err)
	assert.Equal(t, relay1.RequestCounter["builder_getHeaderV1"], 1)
	assert.Equal(t, relay2.RequestCounter["builder_getHeaderV1"], 1)
	assert.Equal(t, "12345678", res.Message.Value.String())

	// ---
	// Test with a slow relay - ensuring that a specific GetHeaderTimeout is respected.
	// relay2 responds with a delay longer than GetHeaderTimeout. Therefore only response from relay1 is used.
	// ---
	server2, err := newTestBoostRPCServerWithTimeout([]string{relay1.URL, relay2.URL}, time.Millisecond*100)
	require.Nil(t, err, err)

	client2 := gethRpc.DialInProc(server2)
	defer client2.Close()

	relay2.SetHandler("builder_getHeaderV1", makeBuilderGetHeaderV1Handler(big.NewInt(12345678), 110*time.Millisecond))
	err = client2.Call(&res, "builder_getHeaderV1", parentHash.Hex())
	require.Nil(t, err, err)
	assert.Equal(t, relay1.RequestCounter["builder_getHeaderV1"], 2)
	assert.Equal(t, relay2.RequestCounter["builder_getHeaderV1"], 2)
	assert.Equal(t, "12345", res.Message.Value.String())
}

func TestE2E_GetHeaderError(t *testing.T) {
	relay1 := setupMockRelay()
	server, err := newTestBoostRPCServer([]string{relay1.URL})
	require.Nil(t, err, err)
	defer server.Stop()

	client := gethRpc.DialInProc(server)
	defer client.Close()

	res := new(types.GetHeaderResponse)
	err = client.Call(&res, "builder_getHeaderV1", nil)
	require.Error(t, err)
	require.Equal(t, err.Error(), errNoBlockHash.Error())
}

func TestE2E_GetPayload(t *testing.T) {
	relay1, relay2 := setupMockRelay(), setupMockRelay()
	server, err := newTestBoostRPCServer([]string{relay1.URL, relay2.URL})
	require.Nil(t, err, err)
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
	err = client.Call(&res, "builder_getPayloadV1", "0x0000000000000000000000000000000000000000000000000000000000000001")
	require.Nil(t, err, err)
	assert.Equal(t, relay1.RequestCounter["builder_getPayloadV1"], 1)
	assert.Equal(t, relay2.RequestCounter["builder_getPayloadV1"], 1)
}
