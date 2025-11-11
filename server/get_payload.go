package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"slices"
	"strconv"
	"sync/atomic"
	"time"

	builderApi "github.com/attestantio/go-builder-client/api"
	builderApiDeneb "github.com/attestantio/go-builder-client/api/deneb"
	builderApiFulu "github.com/attestantio/go-builder-client/api/fulu"
	eth2Api "github.com/attestantio/go-eth2-client/api"
	eth2ApiV1Bellatrix "github.com/attestantio/go-eth2-client/api/v1/bellatrix"
	eth2ApiV1Capella "github.com/attestantio/go-eth2-client/api/v1/capella"
	eth2ApiV1Deneb "github.com/attestantio/go-eth2-client/api/v1/deneb"
	eth2ApiV1Electra "github.com/attestantio/go-eth2-client/api/v1/electra"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/flashbots/mev-boost/common"
	"github.com/flashbots/mev-boost/config"
	"github.com/flashbots/mev-boost/server/params"
	"github.com/flashbots/mev-boost/server/types"
	"github.com/sirupsen/logrus"
)

var (
	errInvalidVersion   = errors.New("invalid version")
	errEmptyPayload     = errors.New("empty payload")
	errInvalidBlockhash = errors.New("invalid blockhash")
	errInvalidKZGLength = errors.New("invalid KZG commitments length")
	errInvalidKZG       = errors.New("invalid KZG commitment")
	errFailedToConvert  = errors.New("failed to convert block from SSZ to JSON")
)

type GetPayloadVersion string

const (
	GetPayloadV1 GetPayloadVersion = "V1"
	GetPayloadV2 GetPayloadVersion = "V2"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Core Logic
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type payloadResult struct {
	success  bool
	response *builderApi.VersionedSubmitBlindedBlockResponse
}

// Deprecated: For reference: https://github.com/ethereum/builder-specs/issues/119
// getPayload requests the payload (execution payload, blobs bundle, etc) from the relays
func (m *BoostService) getPayload(log *logrus.Entry, signedBlindedBeaconBlockBytes []byte, userAgent, proposerContentType, proposerAcceptContentTypes, proposerEthConsensusVersion string) (payloadResult, bidResp) {
	result, bid := m.innerGetPayload(log, signedBlindedBeaconBlockBytes, userAgent, proposerContentType, proposerAcceptContentTypes, proposerEthConsensusVersion, GetPayloadV1)
	return result, bid
}

// getPayloadV2 submits the signed blinded beacon block to relays for submission without returning the payload and blobs
func (m *BoostService) getPayloadV2(log *logrus.Entry, signedBlindedBeaconBlockBytes []byte, userAgent, proposerContentType, proposerAcceptContentTypes, proposerEthConsensusVersion string) (payloadResult, bidResp) {
	result, bid := m.innerGetPayload(log, signedBlindedBeaconBlockBytes, userAgent, proposerContentType, proposerAcceptContentTypes, proposerEthConsensusVersion, GetPayloadV2)
	return result, bid
}

func (m *BoostService) innerGetPayload(log *logrus.Entry, signedBlindedBeaconBlockBytes []byte, userAgent, proposerContentType, proposerAcceptContentTypes, proposerEthConsensusVersion string, version GetPayloadVersion) (payloadResult, bidResp) {
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
		return payloadResult{}, bidResp{}
	}

	// Get information about the request
	slot, err := request.Slot()
	if err != nil {
		log.WithError(err).Error("failed to get request slot")
		return payloadResult{}, bidResp{}
	}
	blockHash, err := request.ExecutionBlockHash()
	if err != nil {
		log.WithError(err).Error("failed to get request block hash")
		return payloadResult{}, bidResp{}
	}
	parentHash, err := request.ExecutionParentHash()
	if err != nil {
		log.WithError(err).Error("failed to get request parent hash")
		return payloadResult{}, bidResp{}
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
		log.Warn("no bid for this payload found, was getHeader called before?")
	} else if len(originalBid.relays) == 0 {
		log.Warn("bid found but no associated relays")
	}

	// Prepare for requests
	resultCh := make(chan payloadResult, len(m.relays))
	var received atomic.Bool
	go func() {
		// Make sure we receive a response within the timeout
		time.Sleep(m.httpClientGetPayload.Timeout)
		resultCh <- payloadResult{}
	}()

	// Create a context with a timeout as configured in the http client
	requestCtx, requestCtxCancel := context.WithTimeout(context.Background(), m.httpClientGetPayload.Timeout)
	defer requestCtxCancel()

	for _, relay := range m.relays {
		go func(relay types.RelayEntry, versionToUse GetPayloadVersion) {
			var url string
			if versionToUse == GetPayloadV1 {
				url = relay.GetURI(params.PathGetPayload)
			} else {
				url = relay.GetURI(params.PathGetPayloadV2)
			}
			innerLog := log.WithField("url", url)

			// If the request fails, try again a few times with 100ms between tries
			resp, err := retry(requestCtx, m.requestMaxRetries, 100*time.Millisecond, func() (*http.Response, error) {
				innerLog = innerLog.WithField("url", url)
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
				innerLog.WithField("relaySupportsSSZ", relaySupportsSSZ).Debug("encoding preference")

				// If the relay provided the bid in JSON or did not provide a bid for this payload,
				// we must convert the signed blinded beacon block from SSZ to JSON for this relay
				if parsedProposerContentType == MediaTypeOctetStream && !relaySupportsSSZ {
					requestContentType = MediaTypeJSON
					startTime := time.Now()
					requestBytes, err = convertSSZToJSON(proposerEthConsensusVersion, signedBlindedBeaconBlockBytes)
					if err != nil {
						innerLog.WithError(errFailedToConvert).Error("failed to convert SSZ to JSON")
						return nil, err
					}
					innerLog.WithFields(logrus.Fields{
						"relayProvidedBid": slices.Contains(originalBid.relays, relay),
						"conversionTime":   time.Since(startTime),
					}).Info("Converted request from SSZ to JSON for relay")
				}

				innerLog.WithFields(logrus.Fields{
					"version": versionToUse,
				}).Info("calling getPayload")

				// Make a new request
				req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, url, bytes.NewReader(requestBytes))
				if err != nil {
					innerLog.WithError(err).Warn("error creating new request")
					return nil, err
				}

				// Add header fields to this request
				req.Header.Set(HeaderAccept, proposerAcceptContentTypes)
				req.Header.Set(HeaderContentType, requestContentType)
				req.Header.Set(HeaderEthConsensusVersion, proposerEthConsensusVersion)
				req.Header.Set(HeaderKeySlotUID, currentSlotUID)
				req.Header.Set(HeaderDateMilliseconds, fmt.Sprintf("%d", time.Now().UTC().UnixMilli()))
				req.Header.Set(HeaderUserAgent, userAgent)

				statusCode := http.StatusOK
				endpoint := params.PathGetPayload
				if versionToUse == GetPayloadV2 {
					statusCode = http.StatusAccepted
					endpoint = params.PathGetPayloadV2
				}
				// Send the request and record latency
				innerLog.Debug("submitting signed blinded block")
				start := time.Now()
				resp, err := m.httpClientGetPayload.Do(req)
				RecordRelayLatency(endpoint, relay.String(), float64(time.Since(start).Microseconds()))
				if err != nil {
					innerLog.WithError(err).Warnf("error calling getPayload%s on relay", versionToUse)
					return nil, err
				}

				RecordRelayStatusCode(strconv.Itoa(statusCode), endpoint, relay.String())
				// Check that the response was successful

				// If the response status code doesn't match expected, read error body once
				if resp.StatusCode != statusCode {
					errorBody, err := io.ReadAll(resp.Body)
					if err != nil {
						innerLog.WithError(err).Warn("error reading error body")
						return nil, err
					}
					errorBodyStr := string(errorBody)
					innerLog.WithField("errorBody", errorBodyStr).Warnf("unexpected status code %d", resp.StatusCode)

					// If the relay does not support V2 API, retry with V1 API
					// we can fallback to V1 API if the status code returned >= 400. There is no harm
					// falling back to the V1 API, falling back to the V1 API in the case of any error
					// can be beneficial to the proposer to avoid a missed slot.
					if resp.StatusCode >= http.StatusBadRequest && url == relay.GetURI(params.PathGetPayloadV2) {
						innerLog.Warn("relay may not support getPayloadV2 endpoint, retrying with getPayloadV1 endpoint")
						// retry with v1 api
						url = relay.GetURI(params.PathGetPayload)
						versionToUse = GetPayloadV1
						return nil, errRetryWithV1API
					}

					err = fmt.Errorf("%w: %d: %s", errHTTPErrorResponse, resp.StatusCode, errorBodyStr)
					return nil, err
				}

				return resp, nil
			})
			if err != nil {
				innerLog.WithError(err).Warn("failed to submit signed blinded block after retries")
				return
			}
			defer resp.Body.Close()

			var result payloadResult
			result.success = true

			if versionToUse == GetPayloadV1 {
				// Get the resp body content
				respBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					innerLog.WithError(err).Warn("error reading response body")
					return
				}

				// Get the response's content type
				respContentType, _, err := mime.ParseMediaType(resp.Header.Get(HeaderContentType))
				if err != nil {
					innerLog.WithError(err).Warn("error parsing response content type")
					respContentType = MediaTypeJSON
				}
				innerLog = innerLog.WithField("respContentType", respContentType)

				// Get the response's eth consensus version
				respEthConsensusVersion := resp.Header.Get(HeaderEthConsensusVersion)
				innerLog = innerLog.WithField("respEthConsensusVersion", respEthConsensusVersion)

				// Decode response
				response := new(builderApi.VersionedSubmitBlindedBlockResponse)
				err = decodeSubmitBlindedBlockResponse(respBytes, respContentType, respEthConsensusVersion, response)
				if err != nil {
					innerLog.WithError(err).Warn("error decoding bid")
					return
				}

				// Check that the payload matches our request
				err = verifyPayload(innerLog, request, response)
				if err != nil {
					innerLog.WithError(err).Warn("error verifying payload")
					return
				}

				result.response = response
			}

			// We have received a valid response, return the first one.
			// The other requests will be running in the background to provide redundancy
			// in case the relay provider which returned the first request fails to broadcast the block.
			if received.CompareAndSwap(false, true) {
				resultCh <- result
				log.Info("successfully submitted blinded block to relay")
			} else {
				log.Trace("discarding response, already received a correct response")
			}
		}(relay, version)
	}

	// Wait for the first request to complete
	return <-resultCh, originalBid
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Verification Functions
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// verifyPayload checks that the payload is valid
func verifyPayload(log *logrus.Entry, request *eth2Api.VersionedSignedBlindedBeaconBlock, response *builderApi.VersionedSubmitBlindedBlockResponse) error {
	// Verify that request & response versions are the same
	if request.Version != response.Version {
		log.WithFields(logrus.Fields{
			"requestVersion":  request.Version,
			"responseVersion": response.Version,
		}).Error("response version does not match request version")
		return errInvalidVersion
	}

	// Verify payload is not empty
	if getPayloadResponseIsEmpty(response) {
		log.Error("response with empty data!")
		return errEmptyPayload
	}

	// Verify that the request & response block hashes are the same
	err := verifyBlockHash(log, request, response)
	if err != nil {
		log.WithError(err).Error("requestBlockHash does not equal responseBlockHash")
		return err
	}

	// Verify that the request & response blobs bundle is the same
	if request.Version >= spec.DataVersionDeneb {
		err = verifyBlobsBundle(log, request, response)
		if err != nil {
			log.WithError(err).Error("requestBlockHash does not equal responseBlockHash")
			return err
		}
	}

	return nil
}

// verifyBlockHash checks that the block hash is correct
func verifyBlockHash(log *logrus.Entry, request *eth2Api.VersionedSignedBlindedBeaconBlock, response *builderApi.VersionedSubmitBlindedBlockResponse) error {
	// Get the request's block hash
	requestBlockHash, err := request.ExecutionBlockHash()
	if err != nil {
		log.WithError(err).Error("failed to get request block hash")
		return err
	}

	// Get the response's block hash
	responseBlockHash, err := response.BlockHash()
	if err != nil {
		log.WithError(err).Error("failed to get response block hash")
		return err
	}

	// Verify that they're the same
	if requestBlockHash != responseBlockHash {
		log.WithFields(logrus.Fields{
			"responseBlockHash": responseBlockHash.String(),
		}).Error("requestBlockHash does not equal responseBlockHash")
		return errInvalidBlockhash
	}

	return nil
}

// verifyBlobsBundle checks that blobs bundle is correct
func verifyBlobsBundle(log *logrus.Entry, request *eth2Api.VersionedSignedBlindedBeaconBlock, response *builderApi.VersionedSubmitBlindedBlockResponse) error {
	// Get the request's blob KZG commitments
	requestCommitments, err := request.BlobKZGCommitments()
	if err != nil {
		log.WithError(err).Error("failed to get request commitments")
		return err
	}

	// Get the response's blobs bundle
	responseBlobsBundle, err := response.BlobsBundle()
	if err != nil {
		log.WithError(err).Error("failed to get response blobs bundle")
		return err
	}

	// Check blobs
	responseBlobs, err := responseBlobsBundle.Blobs()
	if err != nil {
		log.WithError(err).Error("failed to get response blobs")
		return err
	}
	if len(requestCommitments) != len(responseBlobs) {
		log.WithFields(logrus.Fields{
			"requestBlobCommitments": len(requestCommitments),
			"responseBlobs":          len(responseBlobs),
		}).Error("wrong lengths for blobs")
		return errInvalidKZGLength
	}

	// Check commitments
	responseCommitments, err := responseBlobsBundle.Commitments()
	if err != nil {
		log.WithError(err).Error("failed to get response commitments")
		return err
	}
	if len(requestCommitments) != len(responseCommitments) {
		log.WithFields(logrus.Fields{
			"requestBlobCommitments": len(requestCommitments),
			"responseCommitments":    len(responseCommitments),
		}).Error("wrong lengths for commitments")
		return errInvalidKZGLength
	}
	for i, commitment := range requestCommitments {
		if commitment != responseCommitments[i] {
			log.WithFields(logrus.Fields{
				"index":                  i,
				"requestBlobCommitment":  commitment.String(),
				"responseBlobCommitment": responseCommitments[i].String(),
			}).Error("requestBlobCommitment does not equal responseBlobCommitment")
			return errInvalidKZG
		}
	}

	// Check proofs
	responseProofs, err := responseBlobsBundle.Proofs()
	if err != nil {
		log.WithError(err).Error("failed to get response proofs")
		return err
	}

	if request.Version >= spec.DataVersionFulu {
		if len(requestCommitments)*common.CellsPerExtBlob != len(responseProofs) {
			log.WithFields(logrus.Fields{
				"requestBlobCommitments": len(requestCommitments),
				"responseProofs":         len(responseProofs),
				"cellsPerExtBlob":        common.CellsPerExtBlob,
			}).Error("wrong lengths for proofs")
			return errInvalidKZGLength
		}
	} else {
		if len(requestCommitments) != len(responseProofs) {
			log.WithFields(logrus.Fields{
				"requestBlobCommitments": len(requestCommitments),
				"responseProofs":         len(responseProofs),
			}).Error("wrong lengths for proofs")
			return errInvalidKZGLength
		}
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Serialization Functions
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// canUnmarshalSSZ is an interface for types that can unmarshal SSZ
type canUnmarshalSSZ interface {
	UnmarshalSSZ(input []byte) error
}

// convertSSZToJSON converts SSZ-encoded bytes to JSON based on the given ethConsensusVersion
func convertSSZToJSON(ethConsensusVersion string, sszBytes []byte) ([]byte, error) {
	var block canUnmarshalSSZ
	switch ethConsensusVersion {
	case EthConsensusVersionBellatrix:
		block = new(eth2ApiV1Bellatrix.SignedBlindedBeaconBlock)
	case EthConsensusVersionCapella:
		block = new(eth2ApiV1Capella.SignedBlindedBeaconBlock)
	case EthConsensusVersionDeneb:
		block = new(eth2ApiV1Deneb.SignedBlindedBeaconBlock)
	case EthConsensusVersionElectra:
		block = new(eth2ApiV1Electra.SignedBlindedBeaconBlock)
	case EthConsensusVersionFulu:
		block = new(eth2ApiV1Electra.SignedBlindedBeaconBlock)
	default:
		return nil, errInvalidForkVersion
	}

	// Unmarshal the SSZ-encoded bytes into the block
	if err := block.UnmarshalSSZ(sszBytes); err != nil {
		return nil, err
	}

	// Re-encode the block as JSON
	return json.Marshal(block)
}

// decodeSignedBlindedBeaconBlock will decode the request block in either JSON or SSZ.
// Note: when decoding JSON, we must attempt decoding from newest to oldest fork version.
func decodeSignedBlindedBeaconBlock(in []byte, contentType, ethConsensusVersion string, out *eth2Api.VersionedSignedBlindedBeaconBlock) error {
	switch contentType {
	case MediaTypeOctetStream:
		if ethConsensusVersion == "" {
			return types.ErrMissingEthConsensusVersion
		}
		switch ethConsensusVersion {
		case EthConsensusVersionBellatrix:
			out.Version = spec.DataVersionBellatrix
			out.Bellatrix = new(eth2ApiV1Bellatrix.SignedBlindedBeaconBlock)
			return out.Bellatrix.UnmarshalSSZ(in)
		case EthConsensusVersionCapella:
			out.Version = spec.DataVersionCapella
			out.Capella = new(eth2ApiV1Capella.SignedBlindedBeaconBlock)
			return out.Capella.UnmarshalSSZ(in)
		case EthConsensusVersionDeneb:
			out.Version = spec.DataVersionDeneb
			out.Deneb = new(eth2ApiV1Deneb.SignedBlindedBeaconBlock)
			return out.Deneb.UnmarshalSSZ(in)
		case EthConsensusVersionElectra:
			out.Version = spec.DataVersionElectra
			out.Electra = new(eth2ApiV1Electra.SignedBlindedBeaconBlock)
			return out.Electra.UnmarshalSSZ(in)
		case EthConsensusVersionFulu:
			out.Version = spec.DataVersionFulu
			out.Fulu = new(eth2ApiV1Electra.SignedBlindedBeaconBlock)
			return out.Fulu.UnmarshalSSZ(in)
		default:
			return errInvalidForkVersion
		}
	case MediaTypeJSON:
		if ethConsensusVersion == "" {
			return types.ErrMissingEthConsensusVersion
		}
		var err error
		switch ethConsensusVersion {
		case EthConsensusVersionBellatrix:
			block := new(eth2ApiV1Bellatrix.SignedBlindedBeaconBlock)
			if err = json.Unmarshal(in, block); err != nil {
				return err
			}
			out.Version = spec.DataVersionBellatrix
			out.Bellatrix = block
		case EthConsensusVersionCapella:
			block := new(eth2ApiV1Capella.SignedBlindedBeaconBlock)
			if err = json.Unmarshal(in, block); err != nil {
				return err
			}
			out.Version = spec.DataVersionCapella
			out.Capella = block
		case EthConsensusVersionDeneb:
			block := new(eth2ApiV1Deneb.SignedBlindedBeaconBlock)
			if err = json.Unmarshal(in, block); err != nil {
				return err
			}
			out.Version = spec.DataVersionDeneb
			out.Deneb = block
		case EthConsensusVersionElectra:
			block := new(eth2ApiV1Electra.SignedBlindedBeaconBlock)
			if err = json.Unmarshal(in, block); err != nil {
				return err
			}
			out.Version = spec.DataVersionElectra
			out.Electra = block
		case EthConsensusVersionFulu:
			block := new(eth2ApiV1Electra.SignedBlindedBeaconBlock)
			if err = json.Unmarshal(in, block); err != nil {
				return err
			}
			out.Version = spec.DataVersionFulu
			out.Fulu = block
		default:
			return errInvalidForkVersion
		}
		return nil
	}
	return types.ErrInvalidContentType
}

// decodeSubmitBlindedBlockResponse will decode the response contents in either JSON or SSZ
func decodeSubmitBlindedBlockResponse(in []byte, contentType, ethConsensusVersion string, out *builderApi.VersionedSubmitBlindedBlockResponse) error {
	switch contentType {
	case MediaTypeOctetStream:
		if ethConsensusVersion == "" {
			return types.ErrMissingEthConsensusVersion
		}
		switch ethConsensusVersion {
		case EthConsensusVersionBellatrix:
			out.Version = spec.DataVersionBellatrix
			out.Bellatrix = new(bellatrix.ExecutionPayload)
			return out.Bellatrix.UnmarshalSSZ(in)
		case EthConsensusVersionCapella:
			out.Version = spec.DataVersionCapella
			out.Capella = new(capella.ExecutionPayload)
			return out.Capella.UnmarshalSSZ(in)
		case EthConsensusVersionDeneb:
			out.Version = spec.DataVersionDeneb
			out.Deneb = new(builderApiDeneb.ExecutionPayloadAndBlobsBundle)
			return out.Deneb.UnmarshalSSZ(in)
		case EthConsensusVersionElectra:
			out.Version = spec.DataVersionElectra
			out.Electra = new(builderApiDeneb.ExecutionPayloadAndBlobsBundle)
			return out.Electra.UnmarshalSSZ(in)
		case EthConsensusVersionFulu:
			out.Version = spec.DataVersionFulu
			out.Fulu = new(builderApiFulu.ExecutionPayloadAndBlobsBundle)
			return out.Fulu.UnmarshalSSZ(in)
		default:
			return errInvalidForkVersion
		}
	case MediaTypeJSON:
		return json.Unmarshal(in, out)
	}
	return types.ErrInvalidContentType
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Response Functions
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// respondGetPayloadJSON responds to the proposer in JSON
func (m *BoostService) respondGetPayloadJSON(w http.ResponseWriter, result *builderApi.VersionedSubmitBlindedBlockResponse) {
	w.Header().Set(HeaderContentType, MediaTypeJSON)
	w.WriteHeader(http.StatusOK)

	// Serialize and write the data
	if err := json.NewEncoder(w).Encode(&result); err != nil {
		m.log.WithField("response", result).WithError(err).Error("could not write OK response")
		http.Error(w, "", http.StatusInternalServerError)
	}
}

// respondGetPayloadSSZ responds to the proposer in SSZ
func (m *BoostService) respondGetPayloadSSZ(w http.ResponseWriter, result *builderApi.VersionedSubmitBlindedBlockResponse) {
	// Serialize the response
	var err error
	var sszData []byte
	switch result.Version {
	case spec.DataVersionBellatrix:
		w.Header().Set(HeaderEthConsensusVersion, EthConsensusVersionBellatrix)
		sszData, err = result.Bellatrix.MarshalSSZ()
	case spec.DataVersionCapella:
		w.Header().Set(HeaderEthConsensusVersion, EthConsensusVersionCapella)
		sszData, err = result.Capella.MarshalSSZ()
	case spec.DataVersionDeneb:
		w.Header().Set(HeaderEthConsensusVersion, EthConsensusVersionDeneb)
		sszData, err = result.Deneb.MarshalSSZ()
	case spec.DataVersionElectra:
		w.Header().Set(HeaderEthConsensusVersion, EthConsensusVersionElectra)
		sszData, err = result.Electra.MarshalSSZ()
	case spec.DataVersionFulu:
		w.Header().Set(HeaderEthConsensusVersion, EthConsensusVersionFulu)
		sszData, err = result.Fulu.MarshalSSZ()
	case spec.DataVersionUnknown, spec.DataVersionPhase0, spec.DataVersionAltair:
		err = errInvalidForkVersion
	}
	if err != nil {
		m.log.WithError(err).Error("error serializing response as SSZ")
		http.Error(w, "failed to serialize response", http.StatusInternalServerError)
		return
	}

	// Write the header
	w.Header().Set(HeaderContentType, MediaTypeOctetStream)
	w.WriteHeader(http.StatusOK)

	// Write SSZ data
	if _, err := w.Write(sszData); err != nil {
		m.log.WithError(err).Error("error writing SSZ response")
		http.Error(w, "failed to write response", http.StatusInternalServerError)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Other Functions
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// bidKey makes a map key for a specific bid
func bidKey(slot phase0.Slot, blockHash phase0.Hash32) string {
	return fmt.Sprintf("%v%v", slot, blockHash)
}

// retry executes the provided function until it succeeds, the context is done, or
// the maximum number of attempts is reached. It waits for 'delay' between attempts.
func retry(ctx context.Context, maxAttempts int, delay time.Duration, fn func() (*http.Response, error)) (*http.Response, error) {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Check context before starting an attempt
		err := ctx.Err()
		if err != nil {
			return nil, err
		}

		// Execute the function
		resp, err := fn()
		if err == nil {
			return resp, nil
		}

		// Save the last error
		lastErr = err

		// Wait for the delay before retrying, unless context is done
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}
	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
