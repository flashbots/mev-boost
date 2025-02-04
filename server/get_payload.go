package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	builderApi "github.com/attestantio/go-builder-client/api"
	denebApi "github.com/attestantio/go-builder-client/api/deneb"
	eth2ApiV1Bellatrix "github.com/attestantio/go-eth2-client/api/v1/bellatrix"
	eth2ApiV1Capella "github.com/attestantio/go-eth2-client/api/v1/capella"
	eth2ApiV1Deneb "github.com/attestantio/go-eth2-client/api/v1/deneb"
	eth2ApiV1Electra "github.com/attestantio/go-eth2-client/api/v1/electra"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/flashbots/mev-boost/config"
	"github.com/flashbots/mev-boost/server/params"
	"github.com/flashbots/mev-boost/server/types"
	"github.com/sirupsen/logrus"
)

type Payload interface {
	*eth2ApiV1Bellatrix.SignedBlindedBeaconBlock |
		*eth2ApiV1Capella.SignedBlindedBeaconBlock |
		*eth2ApiV1Deneb.SignedBlindedBeaconBlock |
		*eth2ApiV1Electra.SignedBlindedBeaconBlock
}

var (
	errInvalidVersion   = errors.New("invalid version")
	errEmptyPayload     = errors.New("empty payload")
	errInvalidBlockhash = errors.New("invalid blockhash")
	errInvalidKZGLength = errors.New("invalid KZG commitments length")
	errInvalidKZG       = errors.New("invalid KZG commitment")
)

// processPayload requests the payload (execution payload, blobs bundle, etc) from the relays
func processPayload[P Payload](m *BoostService, log *logrus.Entry, ua UserAgent, blindedBlock P) (*builderApi.VersionedSubmitBlindedBlockResponse, bidResp) {
	var (
		slot      = slot(blindedBlock)
		blockHash = blockHash(blindedBlock)
	)

	// Get the currentSlotUID for this slot
	currentSlotUID := ""
	m.slotUIDLock.Lock()
	if m.slotUID.slot == slot {
		currentSlotUID = m.slotUID.uid.String()
	} else {
		log.Warnf("latest slotUID is for slot %d rather than payload slot %d", m.slotUID.slot, slot)
	}
	m.slotUIDLock.Unlock()

	// Prepare logger
	log = prepareLogger(log, blindedBlock, ua, currentSlotUID)

	// Log how late into the slot the request starts
	slotStartTimestamp := m.genesisTime + uint64(slot)*config.SlotTimeSec
	msIntoSlot := uint64(time.Now().UTC().UnixMilli()) - slotStartTimestamp*1000
	log.WithFields(logrus.Fields{
		"genesisTime": m.genesisTime,
		"slotTimeSec": config.SlotTimeSec,
		"msIntoSlot":  msIntoSlot,
	}).Infof("submitBlindedBlock request start - %d milliseconds into slot %d", msIntoSlot, slot)

	// Get the bid!
	m.bidsLock.Lock()
	originalBid := m.bids[bidKey(slot, blockHash)]
	m.bidsLock.Unlock()
	if originalBid.response.IsEmpty() {
		log.Error("no bid for this getPayload payload found, was getHeader called before?")
	} else if len(originalBid.relays) == 0 {
		log.Warn("bid found but no associated relays")
	}

	// Add request headers
	headers := map[string]string{
		HeaderKeySlotUID:      currentSlotUID,
		HeaderStartTimeUnixMS: fmt.Sprintf("%d", time.Now().UTC().UnixMilli()),
	}

	// Prepare for requests
	resultCh := make(chan *builderApi.VersionedSubmitBlindedBlockResponse, len(m.relays))
	var received atomic.Bool
	go func() {
		// Make sure we receive a response within the timeout
		time.Sleep(m.httpClientGetPayload.Timeout)
		resultCh <- nil
	}()

	// Prepare the request context, which will be cancelled after the first successful response from a relay
	requestCtx, requestCtxCancel := context.WithCancel(context.Background())
	defer requestCtxCancel()

	for _, relay := range m.relays {
		go func(relay types.RelayEntry) {
			url := relay.GetURI(params.PathGetPayload)
			log := log.WithField("url", url)
			log.Debug("calling getPayload")

			responsePayload := new(builderApi.VersionedSubmitBlindedBlockResponse)
			_, err := SendHTTPRequestWithRetries(requestCtx, m.httpClientGetPayload, http.MethodPost, url, ua, headers, blindedBlock, responsePayload, m.requestMaxRetries, log)
			if err != nil {
				if errors.Is(requestCtx.Err(), context.Canceled) {
					// This is expected if the payload has already been received by another relay
					log.Info("request was cancelled")
				} else {
					log.WithError(err).Error("error making request to relay")
				}
				return
			}

			if err := verifyPayload(blindedBlock, log, responsePayload); err != nil {
				return
			}

			requestCtxCancel()
			if received.CompareAndSwap(false, true) {
				resultCh <- responsePayload
				log.Info("received payload from relay")
			} else {
				log.Trace("discarding response, already received a correct response")
			}
		}(relay)
	}

	// Wait for the first request to complete
	result := <-resultCh

	return result, originalBid
}

// verifyPayload checks that the payload is valid
func verifyPayload[P Payload](payload P, log *logrus.Entry, response *builderApi.VersionedSubmitBlindedBlockResponse) error {
	// Verify version
	switch any(payload).(type) {
	case *eth2ApiV1Bellatrix.SignedBlindedBeaconBlock:
		if response.Version != spec.DataVersionBellatrix {
			log.WithFields(logrus.Fields{
				"version": response.Version,
			}).Error("response version was not bellatrix")
			return errInvalidVersion
		}
	case *eth2ApiV1Capella.SignedBlindedBeaconBlock:
		if response.Version != spec.DataVersionCapella {
			log.WithFields(logrus.Fields{
				"version": response.Version,
			}).Error("response version was not capella")
			return errInvalidVersion
		}
	case *eth2ApiV1Deneb.SignedBlindedBeaconBlock:
		if response.Version != spec.DataVersionDeneb {
			log.WithFields(logrus.Fields{
				"version": response.Version,
			}).Error("response version was not deneb")
			return errInvalidVersion
		}
	case *eth2ApiV1Electra.SignedBlindedBeaconBlock:
		if response.Version != spec.DataVersionElectra {
			log.WithFields(logrus.Fields{
				"version": response.Version,
			}).Error("response version was not electra")
			return errInvalidVersion
		}
	}

	// Verify payload is not empty
	if getPayloadResponseIsEmpty(response) {
		log.Error("response with empty data!")
		return errEmptyPayload
	}

	// Verify post-conditions
	switch block := any(payload).(type) {
	case *eth2ApiV1Bellatrix.SignedBlindedBeaconBlock:
		if err := verifyBlockHash(log, payload, response.Bellatrix.BlockHash); err != nil {
			return err
		}
	case *eth2ApiV1Capella.SignedBlindedBeaconBlock:
		if err := verifyBlockHash(log, payload, response.Capella.BlockHash); err != nil {
			return err
		}
	case *eth2ApiV1Deneb.SignedBlindedBeaconBlock:
		if err := verifyBlockHash(log, payload, response.Deneb.ExecutionPayload.BlockHash); err != nil {
			return err
		}
		if err := verifyKZGCommitments(log, response.Deneb.BlobsBundle, block.Message.Body.BlobKZGCommitments); err != nil {
			return err
		}
	case *eth2ApiV1Electra.SignedBlindedBeaconBlock:
		if err := verifyBlockHash(log, payload, response.Electra.ExecutionPayload.BlockHash); err != nil {
			return err
		}
		if err := verifyKZGCommitments(log, response.Electra.BlobsBundle, block.Message.Body.BlobKZGCommitments); err != nil {
			return err
		}
	}
	return nil
}

// verifyBlockHash checks that the block hash is correct
func verifyBlockHash[P Payload](log *logrus.Entry, payload P, executionPayloadHash phase0.Hash32) error {
	if blockHash(payload) != executionPayloadHash {
		log.WithFields(logrus.Fields{
			"responseBlockHash": executionPayloadHash.String(),
		}).Error("requestBlockHash does not equal responseBlockHash")
		return errInvalidBlockhash
	}
	return nil
}

// verifyKZGCommitments checks that blobs bundle is valid
func verifyKZGCommitments(log *logrus.Entry, blobs *denebApi.BlobsBundle, commitments []deneb.KZGCommitment) error {
	// Ensure that blobs are valid and matches the request
	if len(commitments) != len(blobs.Blobs) || len(commitments) != len(blobs.Commitments) || len(commitments) != len(blobs.Proofs) {
		log.WithFields(logrus.Fields{
			"requestBlobCommitments":  len(commitments),
			"responseBlobs":           len(blobs.Blobs),
			"responseBlobCommitments": len(blobs.Commitments),
			"responseBlobProofs":      len(blobs.Proofs),
		}).Error("different lengths for blobs/commitments/proofs")
		return errInvalidKZGLength
	}

	for i, commitment := range commitments {
		if commitment != blobs.Commitments[i] {
			log.WithFields(logrus.Fields{
				"index":                  i,
				"requestBlobCommitment":  commitment.String(),
				"responseBlobCommitment": blobs.Commitments[i].String(),
			}).Error("requestBlobCommitment does not equal responseBlobCommitment")
			return errInvalidKZG
		}
	}
	return nil
}

// prepareLogger adds relevant fields to the logger
func prepareLogger[P Payload](log *logrus.Entry, payload P, userAgent UserAgent, slotUID string) *logrus.Entry {
	switch block := any(payload).(type) {
	case *eth2ApiV1Bellatrix.SignedBlindedBeaconBlock:
		return log.WithFields(logrus.Fields{
			"ua":         userAgent,
			"slot":       block.Message.Slot,
			"blockHash":  block.Message.Body.ExecutionPayloadHeader.BlockHash.String(),
			"parentHash": block.Message.Body.ExecutionPayloadHeader.ParentHash.String(),
			"slotUID":    slotUID,
		})
	case *eth2ApiV1Capella.SignedBlindedBeaconBlock:
		return log.WithFields(logrus.Fields{
			"ua":         userAgent,
			"slot":       block.Message.Slot,
			"blockHash":  block.Message.Body.ExecutionPayloadHeader.BlockHash.String(),
			"parentHash": block.Message.Body.ExecutionPayloadHeader.ParentHash.String(),
			"slotUID":    slotUID,
		})
	case *eth2ApiV1Deneb.SignedBlindedBeaconBlock:
		return log.WithFields(logrus.Fields{
			"ua":         userAgent,
			"slot":       block.Message.Slot,
			"blockHash":  block.Message.Body.ExecutionPayloadHeader.BlockHash.String(),
			"parentHash": block.Message.Body.ExecutionPayloadHeader.ParentHash.String(),
			"slotUID":    slotUID,
		})
	case *eth2ApiV1Electra.SignedBlindedBeaconBlock:
		return log.WithFields(logrus.Fields{
			"ua":         userAgent,
			"slot":       block.Message.Slot,
			"blockHash":  block.Message.Body.ExecutionPayloadHeader.BlockHash.String(),
			"parentHash": block.Message.Body.ExecutionPayloadHeader.ParentHash.String(),
			"slotUID":    slotUID,
		})
	}
	return nil
}

// slot returns the block's slot
func slot[P Payload](payload P) phase0.Slot {
	switch block := any(payload).(type) {
	case *eth2ApiV1Bellatrix.SignedBlindedBeaconBlock:
		return block.Message.Slot
	case *eth2ApiV1Capella.SignedBlindedBeaconBlock:
		return block.Message.Slot
	case *eth2ApiV1Deneb.SignedBlindedBeaconBlock:
		return block.Message.Slot
	case *eth2ApiV1Electra.SignedBlindedBeaconBlock:
		return block.Message.Slot
	}
	return 0
}

// blockHash returns the block's hash
func blockHash[P Payload](payload P) phase0.Hash32 {
	switch block := any(payload).(type) {
	case *eth2ApiV1Bellatrix.SignedBlindedBeaconBlock:
		return block.Message.Body.ExecutionPayloadHeader.BlockHash
	case *eth2ApiV1Capella.SignedBlindedBeaconBlock:
		return block.Message.Body.ExecutionPayloadHeader.BlockHash
	case *eth2ApiV1Deneb.SignedBlindedBeaconBlock:
		return block.Message.Body.ExecutionPayloadHeader.BlockHash
	case *eth2ApiV1Electra.SignedBlindedBeaconBlock:
		return block.Message.Body.ExecutionPayloadHeader.BlockHash
	}
	return nilHash
}

// bidKey makes a map key for a specific bid
func bidKey(slot phase0.Slot, blockHash phase0.Hash32) string {
	return fmt.Sprintf("%v%v", slot, blockHash)
}
