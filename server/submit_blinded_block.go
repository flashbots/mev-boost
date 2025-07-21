package server

import (
	"bytes"
	"context"
	"fmt"
	"mime"
	"net/http"
	"slices"
	"sync/atomic"
	"time"

	eth2Api "github.com/attestantio/go-eth2-client/api"
	"github.com/flashbots/mev-boost/config"
	"github.com/flashbots/mev-boost/server/params"
	"github.com/flashbots/mev-boost/server/types"
	"github.com/sirupsen/logrus"
)

// submitBlindedBlock submits the signed blinded beacon block to relays for submission without returning the full payload
func (m *BoostService) submitBlindedBlock(log *logrus.Entry, signedBlindedBeaconBlockBytes []byte, userAgent, proposerContentType, proposerAcceptContentTypes, proposerEthConsensusVersion string) (bool, bidResp) {
	// Get the request's content type
	parsedProposerContentType, _, err := mime.ParseMediaType(proposerContentType)
	if err != nil {
		log.WithError(err).Warn("failed to parse proposer content type")
		parsedProposerContentType = MediaTypeJSON
	}
	log = log.WithField("parsedProposerContentType", parsedProposerContentType)

	// Decode the request
	request := new(eth2Api.VersionedSignedBlindedBeaconBlock)
	err = decodeSignedBlindedBeaconBlock(signedBlindedBeaconBlockBytes, parsedProposerContentType, proposerEthConsensusVersion, request)
	if err != nil {
		log.WithError(err).Error("failed to decode signed blinded beacon block")
		return false, bidResp{}
	}

	// Get information about the request
	slot, err := request.Slot()
	if err != nil {
		log.WithError(err).Error("failed to get request slot")
		return false, bidResp{}
	}
	blockHash, err := request.ExecutionBlockHash()
	if err != nil {
		log.WithError(err).Error("failed to get request block hash")
		return false, bidResp{}
	}
	parentHash, err := request.ExecutionParentHash()
	if err != nil {
		log.WithError(err).Error("failed to get request parent hash")
		return false, bidResp{}
	}

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
	log = log.WithFields(logrus.Fields{
		"slot":       slot,
		"blockHash":  blockHash.String(),
		"parentHash": parentHash.String(),
		"slotUID":    currentSlotUID,
	})

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
		log.Error("no bid for this payload found, was getHeader called before?")
	} else if len(originalBid.relays) == 0 {
		log.Warn("bid found but no associated relays")
	}

	// Prepare for requests
	resultCh := make(chan bool, len(m.relays))
	var received atomic.Bool
	go func() {
		// Make sure we receive a response within the timeout
		time.Sleep(m.httpClientGetPayload.Timeout)
		resultCh <- false
	}()

	// Create a context with a timeout as configured in the http client
	requestCtx, requestCtxCancel := context.WithTimeout(context.Background(), m.httpClientGetPayload.Timeout)
	defer requestCtxCancel()

	for _, relay := range m.relays {
		go func(relay types.RelayEntry) {
			url := relay.GetURI(params.PathSubmitBlindedBlock)
			log := log.WithField("url", url)
			log.Debug("calling submit payload")

			// If the request fails, try again a few times with 100ms between tries
			resp, err := retry(requestCtx, m.requestMaxRetries, 100*time.Millisecond, func() (*http.Response, error) {
				// Default to the content from the proposer
				requestContentType := parsedProposerContentType
				requestBytes := signedBlindedBeaconBlockBytes

				// Check if the relay supports SSZ
				relaySupportsSSZ := false
				for _, originalBidRelay := range originalBid.relays {
					if relay.URL == originalBidRelay.URL {
						relaySupportsSSZ = originalBidRelay.SupportsSSZ
						break
					}
				}
				log.WithField("relaySupportsSSZ", relaySupportsSSZ).Debug("encoding preference")

				// If the relay provided the bid in JSON or did not provide a bid for this payload,
				// we must convert the signed blinded beacon block from SSZ to JSON for this relay
				if parsedProposerContentType == MediaTypeOctetStream && !relaySupportsSSZ {
					requestContentType = MediaTypeJSON
					startTime := time.Now()
					requestBytes, err = convertSSZToJSON(proposerEthConsensusVersion, signedBlindedBeaconBlockBytes)
					if err != nil {
						log.WithError(errFailedToConvert).Error("failed to convert SSZ to JSON")
						return nil, err
					}
					log.WithFields(logrus.Fields{
						"relayProvidedBid": slices.Contains(originalBid.relays, relay),
						"conversionTime":   time.Since(startTime),
					}).Info("Converted request from SSZ to JSON for relay")
				}

				// Make a new request
				req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, url, bytes.NewReader(requestBytes))
				if err != nil {
					log.WithError(err).Warn("error creating new request")
					return nil, err
				}

				// Add header fields to this request
				req.Header.Set(HeaderAccept, proposerAcceptContentTypes)
				req.Header.Set(HeaderContentType, requestContentType)
				req.Header.Set(HeaderEthConsensusVersion, proposerEthConsensusVersion)
				req.Header.Set(HeaderKeySlotUID, currentSlotUID)
				req.Header.Set(HeaderDateMilliseconds, fmt.Sprintf("%d", time.Now().UTC().UnixMilli()))
				req.Header.Set(HeaderUserAgent, userAgent)

				// Send the request
				log.Debug("requesting payload submission")
				resp, err := m.httpClientGetPayload.Do(req)
				if err != nil {
					log.WithError(err).Warn("error calling submit payload on relay")
					return nil, err
				}

				// Check that the response was accepted 202
				if resp.StatusCode != http.StatusAccepted {
					err = fmt.Errorf("%w: %d", errHTTPErrorResponse, resp.StatusCode)
					log.WithError(err).Warn("error status code")
					return nil, err
				}

				return resp, nil
			})
			if err != nil {
				log.WithError(err).Warn("failed to request submit payload after retries")
				return
			}
			defer resp.Body.Close()

			log.Info("relay accepted signed blinded beacon block submission")

			// request is accepted cancel other calls.
			requestCtxCancel()

			// We have received a successful submission, cancel other requests
			if received.CompareAndSwap(false, true) {
				resultCh <- true
				log.Info("successfully submitted blinded block to relay")
			} else {
				log.Trace("discarding response, already received a successful submission")
			}
		}(relay)
	}

	// Wait for the first request to complete
	return <-resultCh, originalBid
}
