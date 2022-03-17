package lib

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// Store stores payloads and retrieves them based on blockHash hashes
type Store interface {
	GetExecutionPayload(blockHash common.Hash) *ExecutionPayloadWithTxRootV1
	SetExecutionPayload(blockHash common.Hash, payload *ExecutionPayloadWithTxRootV1)

	SetForkchoiceResponse(boostPayloadID, relayURL, relayPayloadID string)
	GetForkchoiceResponse(boostPayloadID string) (map[string]string, bool)
}

// map[common.Hash]*ExecutionPayloadWithTxRootV1
// map blockHash to ExecutionPayloadWithTxRootV1. TODO: this has issues, in that blockHash could actually be the same between different payloads
// TODO: clean this up periodically

type store struct {
	payloads     map[common.Hash]*ExecutionPayloadWithTxRootV1
	payloadMutex sync.RWMutex

	forkchoices     map[string]map[string]string // map[boostPayloadID]map[relayURL]relayPayloadID
	forkchoiceMutex sync.RWMutex
}

// NewStore creates an in-mem store
func NewStore() Store {
	return &store{
		payloads:    map[common.Hash]*ExecutionPayloadWithTxRootV1{},
		forkchoices: make(map[string]map[string]string),
	}
}

func (s *store) GetExecutionPayload(blockHash common.Hash) *ExecutionPayloadWithTxRootV1 {
	s.payloadMutex.RLock()
	defer s.payloadMutex.RUnlock()

	payload, ok := s.payloads[blockHash]
	if !ok {
		return nil
	}

	return payload
}

func (s *store) SetExecutionPayload(blockHash common.Hash, payload *ExecutionPayloadWithTxRootV1) {
	if payload == nil {
		return
	}

	s.payloadMutex.Lock()
	defer s.payloadMutex.Unlock()

	s.payloads[blockHash] = payload
}

func (s *store) GetForkchoiceResponse(payloadID string) (map[string]string, bool) {
	s.forkchoiceMutex.RLock()
	defer s.forkchoiceMutex.RUnlock()
	forkchoiceResponses, found := s.forkchoices[payloadID]
	return forkchoiceResponses, found
}

func (s *store) SetForkchoiceResponse(boostPayloadID, relayURL, relayPayloadID string) {
	s.forkchoiceMutex.Lock()
	defer s.forkchoiceMutex.Unlock()
	if _, ok := s.forkchoices[boostPayloadID]; !ok {
		s.forkchoices[boostPayloadID] = make(map[string]string)
	}
	s.forkchoices[boostPayloadID][relayURL] = relayPayloadID
}
