package lib

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// Store stores payloads and retrieves them based on blockHash hashes
type Store interface {
	Get(blockHash common.Hash) *ExecutionPayloadWithTxRootV1
	Set(blockHash common.Hash, payload *ExecutionPayloadWithTxRootV1)
}

// map[common.Hash]*ExecutionPayloadWithTxRootV1
// map blockHash to ExecutionPayloadWithTxRootV1. TODO: this has issues, in that blockHash could actually be the same between different payloads
// TODO: clean this up periodically

type store struct {
	mutex    sync.RWMutex
	payloads map[common.Hash]*ExecutionPayloadWithTxRootV1
}

// NewStore creates an in-mem store
func NewStore() Store {
	return &store{
		payloads: map[common.Hash]*ExecutionPayloadWithTxRootV1{},
	}
}

func (s *store) Get(blockHash common.Hash) *ExecutionPayloadWithTxRootV1 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	payload, ok := s.payloads[blockHash]
	if !ok {
		return nil
	}

	return payload
}

func (s *store) Set(blockHash common.Hash, payload *ExecutionPayloadWithTxRootV1) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if payload == nil {
		return
	}

	s.payloads[blockHash] = payload
}
