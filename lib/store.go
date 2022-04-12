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
	Payload *ExecutionPayloadHeaderV1
	AddedAt time.Time
}

type forkchoiceResponseContainer struct {
	Payload map[string]string // map[relayURL]relayPayloadID
	AddedAt time.Time
}

func newForkchoiceResponseContainer() forkchoiceResponseContainer {
	return forkchoiceResponseContainer{
		Payload: make(map[string]string),
		AddedAt: now(),
	}
}

// Store stores payloads and retrieves them based on blockHash hashes
type Store interface {
	GetExecutionPayload(blockHash common.Hash) *ExecutionPayloadHeaderV1
	SetExecutionPayload(blockHash common.Hash, payload *ExecutionPayloadHeaderV1)

	SetForkchoiceResponse(boostPayloadID, relayURL, relayPayloadID string)
	GetForkchoiceResponse(boostPayloadID string) (map[string]string, bool)

	Cleanup()
}

// map[common.Hash]*ExecutionPayloadHeaderV1
// map blockHash to ExecutionPayloadHeaderV1. TODO: this has issues, in that blockHash could actually be the same between different payloads
// TODO: clean this up periodically

type store struct {
	payloads     map[common.Hash]executionPayloadContainer
	payloadMutex sync.RWMutex

	forkchoices     map[string]forkchoiceResponseContainer // key=boostPayloadID
	forkchoiceMutex sync.RWMutex
}

// NewStore creates an in-mem store. Does not call Store.Cleanup() by default, so memory will build up. Use NewStoreWithCleanup if you want to start a cleanup loop as well.
func NewStore() Store {
	return &store{
		payloads:    make(map[common.Hash]executionPayloadContainer),
		forkchoices: make(map[string]forkchoiceResponseContainer),
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

func (s *store) GetExecutionPayload(blockHash common.Hash) *ExecutionPayloadHeaderV1 {
	s.payloadMutex.RLock()
	defer s.payloadMutex.RUnlock()

	payload, ok := s.payloads[blockHash]
	if !ok {
		return nil
	}

	return payload.Payload
}

func (s *store) SetExecutionPayload(blockHash common.Hash, payload *ExecutionPayloadHeaderV1) {
	if payload == nil {
		return
	}

	s.payloadMutex.Lock()
	defer s.payloadMutex.Unlock()

	s.payloads[blockHash] = executionPayloadContainer{payload, now()}
}

func (s *store) GetForkchoiceResponse(payloadID string) (map[string]string, bool) {
	s.forkchoiceMutex.RLock()
	defer s.forkchoiceMutex.RUnlock()
	forkchoiceResponses, found := s.forkchoices[payloadID]
	return forkchoiceResponses.Payload, found
}

func (s *store) SetForkchoiceResponse(boostPayloadID, relayURL, relayPayloadID string) {
	s.forkchoiceMutex.Lock()
	defer s.forkchoiceMutex.Unlock()
	if _, ok := s.forkchoices[boostPayloadID]; !ok {
		s.forkchoices[boostPayloadID] = newForkchoiceResponseContainer()
	}
	s.forkchoices[boostPayloadID].Payload[relayURL] = relayPayloadID
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

	// Cleanup ForkchoiceResponse
	s.forkchoiceMutex.Lock()
	for entry := range s.forkchoices {
		if time.Since(s.forkchoices[entry].AddedAt) > stateExpiry {
			delete(s.forkchoices, entry)
		}
	}
	s.forkchoiceMutex.Unlock()
}
