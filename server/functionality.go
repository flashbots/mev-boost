package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	builderApi "github.com/attestantio/go-builder-client/api"
	denebApi "github.com/attestantio/go-builder-client/api/deneb"
	builderSpec "github.com/attestantio/go-builder-client/spec"
	eth2ApiV1Deneb "github.com/attestantio/go-eth2-client/api/v1/deneb"
	eth2ApiV1Electra "github.com/attestantio/go-eth2-client/api/v1/electra"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/flashbots/mev-boost/config"
	"github.com/flashbots/mev-boost/server/params"
	"github.com/flashbots/mev-boost/server/types"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Payload interface {
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

func processPayload[P Payload](m *BoostService, log *logrus.Entry, ua UserAgent, blindedBlock P) (*builderApi.VersionedSubmitBlindedBlockResponse, bidResp) {
	var (
		slot      = slot[P](blindedBlock)
		blockHash = blockHash[P](blindedBlock)
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
	log = prepareLogger[P](log, blindedBlock, ua, currentSlotUID)

	// Log how late into the slot the request starts
	slotStartTimestamp := m.genesisTime + slot*config.SlotTimeSec
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
					log.Info("request was cancelled") // this is expected, if payload has already been received by another relay
				} else {
					log.WithError(err).Error("error making request to relay")
				}
				return
			}

			if err := verifyPayload[P](blindedBlock, log, responsePayload); err != nil {
				return
			}

			requestCtxCancel()
			if received.CompareAndSwap(false, true) {
				resultCh <- responsePayload
				log.Info("received payload from relay")
			} else {
				log.Trace("Discarding response, already received a correct response")
			}
		}(relay)
	}

	// Wait for the first request to complete
	result := <-resultCh

	return result, originalBid
}

func verifyPayload[P Payload](payload P, log *logrus.Entry, response *builderApi.VersionedSubmitBlindedBlockResponse) error {
	// Step 1: verify version
	switch any(payload).(type) {
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

	// Step 2: verify payload is not empty
	if getPayloadResponseIsEmpty(response) {
		log.Error("response with empty data!")
		return errEmptyPayload
	}

	// TODO(MariusVanDerWijden): make this generic once
	// execution payload or blobs bundle change between forks.
	var (
		executionPayload *deneb.ExecutionPayload
		blobs            *denebApi.BlobsBundle
	)

	switch any(payload).(type) {
	case *eth2ApiV1Deneb.SignedBlindedBeaconBlock:
		executionPayload = response.Deneb.ExecutionPayload
		blobs = response.Deneb.BlobsBundle
	case *eth2ApiV1Electra.SignedBlindedBeaconBlock:
		executionPayload = response.Electra.ExecutionPayload
		blobs = response.Electra.BlobsBundle
	}

	// Step 3: Ensure the response blockhash matches the request
	if blockHash[P](payload) != executionPayload.BlockHash {
		log.WithFields(logrus.Fields{
			"responseBlockHash": executionPayload.String(),
		}).Error("requestBlockHash does not equal responseBlockHash")
		return errInvalidBlockhash
	}

	// Step 4: Verify KZG commitments
	var commitments []deneb.KZGCommitment
	switch block := any(payload).(type) {
	case *eth2ApiV1Deneb.SignedBlindedBeaconBlock:
		commitments = block.Message.Body.BlobKZGCommitments
	case *eth2ApiV1Electra.SignedBlindedBeaconBlock:
		commitments = block.Message.Body.BlobKZGCommitments
	}
	// Ensure that blobs are valid and matches the request
	if len(commitments) != len(blobs.Blobs) || len(commitments) != len(blobs.Commitments) || len(commitments) != len(blobs.Proofs) {
		log.WithFields(logrus.Fields{
			"requestBlobCommitments":  len(commitments),
			"responseBlobs":           len(blobs.Blobs),
			"responseBlobCommitments": len(blobs.Commitments),
			"responseBlobProofs":      len(blobs.Proofs),
		}).Error("block KZG commitment length does not equal responseBlobs length")
		return errInvalidKZGLength
	}

	for i, commitment := range commitments {
		if commitment != blobs.Commitments[i] {
			log.WithFields(logrus.Fields{
				"requestBlobCommitment":  commitment.String(),
				"responseBlobCommitment": blobs.Commitments[i].String(),
				"index":                  i,
			}).Error("requestBlobCommitment does not equal responseBlobCommitment")
			return errInvalidKZG
		}
	}
	return nil
}

func prepareLogger[P Payload](log *logrus.Entry, payload P, userAgent UserAgent, slotUID string) *logrus.Entry {
	switch block := any(payload).(type) {
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

func slot[P Payload](payload P) uint64 {
	switch block := any(payload).(type) {
	case *eth2ApiV1Deneb.SignedBlindedBeaconBlock:
		return uint64(block.Message.Slot)
	case *eth2ApiV1Electra.SignedBlindedBeaconBlock:
		return uint64(block.Message.Slot)
	}
	return 0
}

func blockHash[P Payload](payload P) phase0.Hash32 {
	switch block := any(payload).(type) {
	case *eth2ApiV1Deneb.SignedBlindedBeaconBlock:
		return block.Message.Body.ExecutionPayloadHeader.BlockHash
	case *eth2ApiV1Electra.SignedBlindedBeaconBlock:
		return block.Message.Body.ExecutionPayloadHeader.BlockHash
	}
	return nilHash
}

func bidKey(slot uint64, blockHash phase0.Hash32) string {
	return fmt.Sprintf("%v%v", slot, blockHash)
}

func (m *BoostService) getHeader(log *logrus.Entry, ua UserAgent, _slot uint64, pubkey, parentHashHex string) (bidResp, error) {
	if len(pubkey) != 98 {
		return bidResp{}, errInvalidPubkey
	}

	if len(parentHashHex) != 66 {
		return bidResp{}, errInvalidHash
	}

	// Make sure we have a uid for this slot
	m.slotUIDLock.Lock()
	if m.slotUID.slot < _slot {
		m.slotUID.slot = _slot
		m.slotUID.uid = uuid.New()
	}
	slotUID := m.slotUID.uid
	m.slotUIDLock.Unlock()
	log = log.WithField("slotUID", slotUID)

	// Log how late into the slot the request starts
	slotStartTimestamp := m.genesisTime + _slot*config.SlotTimeSec
	msIntoSlot := uint64(time.Now().UTC().UnixMilli()) - slotStartTimestamp*1000
	log.WithFields(logrus.Fields{
		"genesisTime": m.genesisTime,
		"slotTimeSec": config.SlotTimeSec,
		"msIntoSlot":  msIntoSlot,
	}).Infof("getHeader request start - %d milliseconds into slot %d", msIntoSlot, _slot)
	// Add request headers
	headers := map[string]string{
		HeaderKeySlotUID:      slotUID.String(),
		HeaderStartTimeUnixMS: fmt.Sprintf("%d", time.Now().UTC().UnixMilli()),
	}
	// Prepare relay responses
	var (
		result = bidResp{}                                 // the final response, containing the highest bid (if any)
		relays = make(map[BlockHashHex][]types.RelayEntry) // relays that sent the bid for a specific blockHash

		mu sync.Mutex
		wg sync.WaitGroup
	)

	// Call the relays
	for _, relay := range m.relays {
		wg.Add(1)
		go func(relay types.RelayEntry) {
			defer wg.Done()
			path := fmt.Sprintf("/eth/v1/builder/header/%d/%s/%s", _slot, parentHashHex, pubkey)
			url := relay.GetURI(path)
			log := log.WithField("url", url)
			responsePayload := new(builderSpec.VersionedSignedBuilderBid)
			code, err := SendHTTPRequest(context.Background(), m.httpClientGetHeader, http.MethodGet, url, ua, headers, nil, responsePayload)
			if err != nil {
				log.WithError(err).Warn("error making request to relay")
				return
			}

			if code == http.StatusNoContent {
				log.Debug("no-content response")
				return
			}

			// Skip if payload is empty
			if responsePayload.IsEmpty() {
				return
			}

			// Getting the bid info will check if there are missing fields in the response
			bidInfo, err := parseBidInfo(responsePayload)
			if err != nil {
				log.WithError(err).Warn("error parsing bid info")
				return
			}

			if bidInfo.blockHash == nilHash {
				log.Warn("relay responded with empty block hash")
				return
			}

			valueEth := weiBigIntToEthBigFloat(bidInfo.value.ToBig())
			log = log.WithFields(logrus.Fields{
				"blockNumber": bidInfo.blockNumber,
				"blockHash":   bidInfo.blockHash.String(),
				"txRoot":      bidInfo.txRoot.String(),
				"value":       valueEth.Text('f', 18),
			})

			if relay.PublicKey.String() != bidInfo.pubkey.String() {
				log.Errorf("bid pubkey mismatch. expected: %s - got: %s", relay.PublicKey.String(), bidInfo.pubkey.String())
				return
			}

			// Verify the relay signature in the relay response
			if !config.SkipRelaySignatureCheck {
				ok, err := checkRelaySignature(responsePayload, m.builderSigningDomain, relay.PublicKey)
				if err != nil {
					log.WithError(err).Error("error verifying relay signature")
					return
				}
				if !ok {
					log.Error("failed to verify relay signature")
					return
				}
			}

			// Verify response coherence with proposer's input data
			if bidInfo.parentHash.String() != parentHashHex {
				log.WithFields(logrus.Fields{
					"originalParentHash": parentHashHex,
					"responseParentHash": bidInfo.parentHash.String(),
				}).Error("proposer and relay parent hashes are not the same")
				return
			}

			isZeroValue := bidInfo.value.IsZero()
			isEmptyListTxRoot := bidInfo.txRoot.String() == "0x7ffe241ea60187fdb0187bfa22de35d1f9bed7ab061d9401fd47e34a54fbede1"
			if isZeroValue || isEmptyListTxRoot {
				log.Warn("ignoring bid with 0 value")
				return
			}
			log.Debug("bid received")

			// Skip if value (fee) is lower than the minimum bid
			if bidInfo.value.CmpBig(m.relayMinBid.BigInt()) == -1 {
				log.Debug("ignoring bid below min-bid value")
				return
			}

			mu.Lock()
			defer mu.Unlock()

			// Remember which relays delivered which bids (multiple relays might deliver the top bid)
			relays[BlockHashHex(bidInfo.blockHash.String())] = append(relays[BlockHashHex(bidInfo.blockHash.String())], relay)

			// Compare the bid with already known top bid (if any)
			if !result.response.IsEmpty() {
				valueDiff := bidInfo.value.Cmp(result.bidInfo.value)
				if valueDiff == -1 { // current bid is less profitable than already known one
					return
				} else if valueDiff == 0 { // current bid is equally profitable as already known one. Use hash as tiebreaker
					previousBidBlockHash := result.bidInfo.blockHash
					if bidInfo.blockHash.String() >= previousBidBlockHash.String() {
						return
					}
				}
			}

			// Use this relay's response as mev-boost response because it's most profitable
			log.Debug("new best bid")
			result.response = *responsePayload
			result.bidInfo = bidInfo
			result.t = time.Now()
		}(relay)
	}
	// Wait for all requests to complete...
	wg.Wait()

	// Set the winning relay before returning
	result.relays = relays[BlockHashHex(result.bidInfo.blockHash.String())]
	return result, nil
}
