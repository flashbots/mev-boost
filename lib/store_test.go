package lib

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func Test_store_Get(t *testing.T) {
	s := NewStore()
	h := common.HexToHash("0x1")
	payload := s.Get(h)
	if payload != nil {
		t.Errorf("Expected nil, got %v", payload)
	}

	payload = &ExecutionPayloadWithTxRootV1{Number: 1}
	s.Set(h, payload)
	if !reflect.DeepEqual(s.Get(h), payload) {
		t.Errorf("Expected %v, got %v", payload, s.Get(h))
	}

	payload = &ExecutionPayloadWithTxRootV1{Number: 2}
	s.Set(h, payload)
	if !reflect.DeepEqual(s.Get(h), payload) {
		t.Errorf("Expected %v, got %v", payload, s.Get(h))
	}
}
