package lib

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

var (
	cleanupLoopInterval = 5 * time.Minute
	stateExpiry         = 7 * time.Minute // a bit more than an epoch with 6.4 min
)

type executionPayloadContainer struct {
	Payload *ExecutionPayloadWithTxRootV1
	AddedAt time.Time
}

type forkchoiceResponseContainer struct {
	Payload map[string]string // map[relayURL]relayPayloadID
	AddedAt time.Time
}

func newForkchoiceResponseContainer() forkchoiceResponseContainer {
	return forkchoiceResponseContainer{
		Payload: make(map[string]string),
		AddedAt: time.Now(),
	}
}

// Store stores payloads and retrieves them based on blockHash hashes
type Store interface {
	GetExecutionPayload(blockHash common.Hash) *ExecutionPayloadWithTxRootV1
	SetExecutionPayload(blockHash common.Hash, payload *ExecutionPayloadWithTxRootV1)

	SetForkchoiceResponse(boostPayloadID, relayURL, relayPayloadID string)
	GetForkchoiceResponse(boostPayloadID string) (map[string]string, bool)

	Cleanup()
}

// map[common.Hash]*ExecutionPayloadWithTxRootV1
// map blockHash to ExecutionPayloadWithTxRootV1. TODO: this has issues, in that blockHash could actually be the same between different payloads
// TODO: clean this up periodically

type store struct {
	payloads     map[common.Hash]executionPayloadContainer
	payloadMutex sync.RWMutex

	forkchoices     map[string]forkchoiceResponseContainer // key=boostPayloadID
	forkchoiceMutex sync.RWMutex
}

// NewStore creates an in-mem store. If startCleanupLoop is true, a goroutine is started that periodically removes old entries.
func NewStore(startCleanupLoop bool) Store {
	s := &store{
		payloads:    make(map[common.Hash]executionPayloadContainer),
		forkchoices: make(map[string]forkchoiceResponseContainer),
	}

	if startCleanupLoop {
		go func() {
			for {
				time.Sleep(cleanupLoopInterval)
				s.Cleanup()
			}
		}()
	}

	return s
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

	s.payloads[blockHash] = executionPayloadContainer{payload, time.Now()}
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
