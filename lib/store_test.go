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

func Test_store_Cleanup(t *testing.T) {
	// Reset 'now' after this test
	defer func() { now = time.Now }()

	s := NewStoreWithCleanup()
	id1 := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")
	id2 := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002")

	// Add a store item 20 minutes in the past
	now = func() time.Time { return time.Now().Add(-20 * time.Minute) }
	s.SetExecutionPayload(id1, &ExecutionPayloadWithTxRootV1{})

	// Add a store item 5 minutes in the past
	now = func() time.Time { return time.Now().Add(-5 * time.Minute) }
	s.SetExecutionPayload(id2, &ExecutionPayloadWithTxRootV1{})

	payload := s.GetExecutionPayload(id1)
	require.NotNil(t, payload)
	payload = s.GetExecutionPayload(id2)
	require.NotNil(t, payload)

	// Cleanup should remove 1 item, because it was added long enough in the past
	s.Cleanup()

	// Test for items
	payload = s.GetExecutionPayload(id1)
	require.Nil(t, payload)
	payload = s.GetExecutionPayload(id2)
	require.NotNil(t, payload)
}
