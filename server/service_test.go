package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flashbots/mev-boost/types"
	"github.com/stretchr/testify/require"
)

func TestWebserverRootHandler(t *testing.T) {
	backend := newTestBackend(t, 1, RelayTimeouts{})

	// Check root handler
	req, _ := http.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	backend.boostRouter.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "hello\n", rr.Body.String())
}

// Example good registerValidator payload
var payloadRegisterValidator = types.RegisterValidatorRequest{
	Message: types.RegisterValidatorRequestMessage{
		FeeRecipient: common.HexToAddress("0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941"),
		Timestamp:    "1234356",
		GasLimit:     "278234191203",
		Pubkey:       _hexToBytes("0xf9716c94aab536227804e859d15207aa7eaaacd839f39dcbdb5adc942842a8d2fb730f9f49fc719fdb86f1873e0ed1c2"),
	},
	Signature: _hexToBytes("0x8682789b16da95ba437a5b51c14ba4e112b50ceacd9730f697c4839b91405280e603fc4367283aa0866af81a21c536c4c452ace2f4146267c5cf6e959955964f4c35f0cedaf80ed99ffc32fe2d28f9390bb30269044fcf20e2dd734c7b287d14"),
}

func TestRegisterValidator(t *testing.T) {
	backend := newTestBackend(t, 1, RelayTimeouts{})
	path := "/registerValidator"

	t.Run("valid request, valid relay response", func(t *testing.T) {
		rr := backend.post(t, path, payloadRegisterValidator)
		require.Equal(t, http.StatusOK, rr.Code)
		require.Equal(t, 1, backend.relays[0].RequestCount[path])
	})

	t.Run("Invalid signature", func(t *testing.T) {
		rr := backend.post(t, path, types.RegisterValidatorRequest{
			Message: types.RegisterValidatorRequestMessage{
				FeeRecipient: common.HexToAddress("0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941"),
				Timestamp:    "1234356",
				Pubkey:       _hexToBytes("0xf9716c94aab536227804e859d15207aa7eaaacd839f39dcbdb5adc942842a8d2fb730f9f49fc719fdb86f1873e0ed1c2"),
			},
			Signature: _hexToBytes("0x8682789b16da95ba437a5b51c14ba4e112b50ceacd9730f697c4839b91405280e603fc4367283aa0866af81a21c536c4c452ace2f4146267c5cf6e959955964f4c35f0cedaf80ed99ffc32fe2d28f9390bb30269044fcf20e2dd734c7b287d"),
		})
		require.Equal(t, http.StatusBadRequest, rr.Code)
		require.Equal(t, errInvalidSignature.Error()+"\n", rr.Body.String())
		require.Equal(t, 1, backend.relays[0].RequestCount[path])
	})

	t.Run("Invalid pubkey", func(t *testing.T) {
		rr := backend.post(t, path, types.RegisterValidatorRequest{
			Message: types.RegisterValidatorRequestMessage{
				FeeRecipient: common.HexToAddress("0xdb65fEd33dc262Fe09D9a2Ba8F80b329BA25f941"),
				Timestamp:    "1234356",
				Pubkey:       _hexToBytes("0xf9716c94aab536227804e859d15207aa7eaaacd839f39dcbdb5adc942842a8d2fb730f9f49fc719fdb86f1873e0ed1"),
			},
			Signature: _hexToBytes("0x8682789b16da95ba437a5b51c14ba4e112b50ceacd9730f697c4839b91405280e603fc4367283aa0866af81a21c536c4c452ace2f4146267c5cf6e959955964f4c35f0cedaf80ed99ffc32fe2d28f9390bb30269044fcf20e2dd734c7b287d14"),
		})
		require.Equal(t, http.StatusBadRequest, rr.Code)
		require.Equal(t, errInvalidPubkey.Error()+"\n", rr.Body.String())
		require.Equal(t, 1, backend.relays[0].RequestCount[path])
	})
}

func TestRegisterValidator_InvalidRelayResponses(t *testing.T) {
	backend := newTestBackend(t, 2, RelayTimeouts{})
	path := "/registerValidator"

	rr := backend.post(t, path, payloadRegisterValidator)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, 1, backend.relays[0].RequestCount[path])
	require.Equal(t, 1, backend.relays[1].RequestCount[path])

	// Now make one relay return an error
	backend.relays[0].HandlerOverride = func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusBadRequest) }
	rr = backend.post(t, path, payloadRegisterValidator)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, 2, backend.relays[0].RequestCount[path])
	require.Equal(t, 2, backend.relays[1].RequestCount[path])

	// Now make both relays return an error - which should cause the request to fail
	backend.relays[1].HandlerOverride = func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusBadRequest) }
	rr = backend.post(t, path, payloadRegisterValidator)
	require.Equal(t, http.StatusBadGateway, rr.Code)
	require.Equal(t, 3, backend.relays[0].RequestCount[path])
	require.Equal(t, 3, backend.relays[1].RequestCount[path])
}