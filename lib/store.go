package lib

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

var (
	cleanupLoopInterval = 5 * time.Minute

	secondsPerSlot = 12
	slotsPerEpoch  = 32
	stateExpiry    = time.Second * time.Duration(secondsPerSlot*slotsPerEpoch*2) // ~2 epochs

	// local now function, used instead of time.Now so it can be overwritten in tests
	now = time.Now
)

type executionPayloadContainer struct {
	Payload *ExecutionPayloadWithTxRootV1
	AddedAt time.Time
}

// Store stores payloads and retrieves them based on blockHash hashes
type Store interface {
	GetExecutionPayload(blockHash common.Hash) *ExecutionPayloadWithTxRootV1
	SetExecutionPayload(blockHash common.Hash, payload *ExecutionPayloadWithTxRootV1)

	Cleanup()
}

// map[common.Hash]*ExecutionPayloadWithTxRootV1
// map blockHash to ExecutionPayloadWithTxRootV1. TODO: this has issues, in that blockHash could actually be the same between different payloads
// TODO: clean this up periodically

type store struct {
	payloads     map[common.Hash]executionPayloadContainer
	payloadMutex sync.RWMutex
}

// NewStore creates an in-mem store. Does not call Store.Cleanup() by default, so memory will build up. Use NewStoreWithCleanup if you want to start a cleanup loop as well.
func NewStore() Store {
	return &store{
		payloads: make(map[common.Hash]executionPayloadContainer),
	}
}

// NewStoreWithCleanup creates an in-mem store, and starts goroutine that periodically removes old entries.
func NewStoreWithCleanup() Store {
	store := NewStore()

	go func() {
		for {
			time.Sleep(cleanupLoopInterval)
			store.Cleanup()
		}
	}()

	return store
}

func (s *store) GetExecutionPayload(blockHash common.Hash) *ExecutionPayloadWithTxRootV1 {
	s.payloadMutex.RLock()
	defer s.payloadMutex.RUnlock()

	payload, ok := s.payloads[blockHash]
	if !ok {
		return nil
	}

	return payload.Payload
}

func (s *store) SetExecutionPayload(blockHash common.Hash, payload *ExecutionPayloadWithTxRootV1) {
	if payload == nil {
		return
	}

	s.payloadMutex.Lock()
	defer s.payloadMutex.Unlock()

	s.payloads[blockHash] = executionPayloadContainer{payload, now()}
}

// Cleanup removes all payloads older than 7 minutes (a bit more than an epoch, which is 6.4 minutes)
func (s *store) Cleanup() {
	// Cleanup ExecutionPayload
	s.payloadMutex.Lock()
	for entry := range s.payloads {
		if time.Since(s.payloads[entry].AddedAt) > stateExpiry {
			delete(s.payloads, entry)
		}
	}
	s.payloadMutex.Unlock()
}
