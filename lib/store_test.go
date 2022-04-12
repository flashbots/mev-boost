package lib

import (
	"reflect"
	"testing"
	"time"

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

	payload = &ExecutionPayloadHeaderV1{BlockNumber: 1}
	s.SetExecutionPayload(h, payload)
	if !reflect.DeepEqual(s.GetExecutionPayload(h), payload) {
		t.Errorf("Expected %v, got %v", payload, s.GetExecutionPayload(h))
	}

	payload = &ExecutionPayloadHeaderV1{BlockNumber: 2}
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

func Test_store_Cleanup(t *testing.T) {
	// Reset 'now' after this test
	defer func() { now = time.Now }()

	s := NewStoreWithCleanup()
	id1 := "123"
	id2 := "234"

	// Add a store item 20 minutes in the past
	now = func() time.Time { return time.Now().Add(-20 * time.Minute) }
	s.SetForkchoiceResponse(id1, "abc", "0x2")

	// Add a store item 5 minutes in the past
	now = func() time.Time { return time.Now().Add(-5 * time.Minute) }
	s.SetForkchoiceResponse(id2, "abc", "0x2")

	_, ok := s.GetForkchoiceResponse(id1)
	require.Equal(t, true, ok)
	_, ok = s.GetForkchoiceResponse(id2)
	require.Equal(t, true, ok)

	// Cleanup should remove 1 item, because it was added long enough in the past
	s.Cleanup()

	// Test for items
	_, ok = s.GetForkchoiceResponse(id1)
	require.Equal(t, false, ok)
	_, ok = s.GetForkchoiceResponse(id2)
	require.Equal(t, true, ok)
}
