package lib

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func Test_store_SetGetExecutionPayload(t *testing.T) {
	s := NewStore()
	h := common.HexToHash("0x1")
	payload := s.GetExecutionPayload(h)
	if payload != nil {
		t.Errorf("Expected nil, got %v", payload)
	}

	payload = &ExecutionPayloadWithTxRootV1{Number: 1}
	s.SetExecutionPayload(h, payload)
	if !reflect.DeepEqual(s.GetExecutionPayload(h), payload) {
		t.Errorf("Expected %v, got %v", payload, s.GetExecutionPayload(h))
	}

	payload = &ExecutionPayloadWithTxRootV1{Number: 2}
	s.SetExecutionPayload(h, payload)
	if !reflect.DeepEqual(s.GetExecutionPayload(h), payload) {
		t.Errorf("Expected %v, got %v", payload, s.GetExecutionPayload(h))
	}
}

func Test_store_SetGetGetForkchoiceResponse(t *testing.T) {
	s := NewStore()
	id := "0x1"
	_, ok := s.GetForkchoiceResponse(id)
	require.Equal(t, false, ok)

	relayURL := "abc"
	relayPayloadID := "0x2"
	s.SetForkchoiceResponse(id, relayURL, relayPayloadID)

	res, ok := s.GetForkchoiceResponse(id)
	require.Equal(t, true, ok)
	require.Equal(t, res[relayURL], relayPayloadID)

	relayURL = "def"
	_, ok = res[relayURL]
	require.Equal(t, false, ok)
	relayPayloadID = "0x3"
	s.SetForkchoiceResponse(id, relayURL, relayPayloadID)

	res, ok = s.GetForkchoiceResponse(id)
	require.Equal(t, true, ok)
	require.Equal(t, res[relayURL], relayPayloadID)
}
