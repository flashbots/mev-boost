package lib

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func Test_store_Get(t *testing.T) {
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
