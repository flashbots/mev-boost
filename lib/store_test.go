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

func Test_store_Cleanup(t *testing.T) {
	// Reset 'now' after this test
	defer func() { now = time.Now }()

	s := NewStoreWithCleanup()
	h1 := common.HexToHash("0x1")
	h2 := common.HexToHash("0x2")
	payload := &ExecutionPayloadHeaderV1{BlockNumber: 1}

	// Add a store item 20 minutes in the past
	now = func() time.Time { return time.Now().Add(-20 * time.Minute) }
	s.SetExecutionPayload(h1, payload)

	// Add a store item 5 minutes in the past
	now = func() time.Time { return time.Now().Add(-5 * time.Minute) }
	s.SetExecutionPayload(h2, payload)

	_payload1 := s.GetExecutionPayload(h1)
	require.NotNil(t, _payload1)
	_payload2 := s.GetExecutionPayload(h2)
	require.NotNil(t, _payload2)

	// Cleanup should remove 1 item, because it was added long enough in the past
	s.Cleanup()

	// Test for items
	_payload1 = s.GetExecutionPayload(h1)
	require.Nil(t, _payload1)
	_payload2 = s.GetExecutionPayload(h2)
	require.NotNil(t, _payload2)
}
