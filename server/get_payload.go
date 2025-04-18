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
	"sync/atomic"
	"time"

	builderApi "github.com/attestantio/go-builder-client/api"
	builderApiDeneb "github.com/attestantio/go-builder-client/api/deneb"
	eth2Api "github.com/attestantio/go-eth2-client/api"
	eth2ApiV1Bellatrix "github.com/attestantio/go-eth2-client/api/v1/bellatrix"
	eth2ApiV1Capella "github.com/attestantio/go-eth2-client/api/v1/capella"
	eth2ApiV1Deneb "github.com/attestantio/go-eth2-client/api/v1/deneb"
	eth2ApiV1Electra "github.com/attestantio/go-eth2-client/api/v1/electra"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
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
	errFailedToDecode   = errors.New("failed to decode payload")
	errFailedToConvert  = errors.New("failed to convert block from SSZ to JSON")
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Core Logic
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// getPayload requests the payload (execution payload, blobs bundle, etc) from the relays
func (m *BoostService) getPayload(log *logrus.Entry, signedBlindedBeaconBlockBytes []byte, userAgent, proposerContentType, proposerAcceptContentTypes, proposerEthConsensusVersion string) (*builderApi.VersionedSubmitBlindedBlockResponse, bidResp) {
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
		return nil, bidResp{}
	}

	// Get information about the request
	slot, err := request.Slot()
	if err != nil {
		log.WithError(err).Error("failed to get request slot")
		return nil, bidResp{}
	}
	blockHash, err := request.ExecutionBlockHash()
	if err != nil {
		log.WithError(err).Error("failed to get request block hash")
		return nil, bidResp{}
	}
	parentHash, err := request.ExecutionParentHash()
	if err != nil {
		log.WithError(err).Error("failed to get request parent hash")
		return nil, bidResp{}
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
		log.Error("no bid for this getPayload payload found, was getHeader called before?")
	} else if len(originalBid.relays) == 0 {
		log.Warn("bid found but no associated relays")
	}

	// Prepare for requests
	resultCh := make(chan *builderApi.VersionedSubmitBlindedBlockResponse, len(m.relays))
	var received atomic.Bool
	go func() {
		// Make sure we receive a response within the timeout
		time.Sleep(m.httpClientGetPayload.Timeout)
		resultCh <- nil
	}()

	// Create a context with a timeout as configured in the http client
	requestCtx, requestCtxCancel := context.WithTimeout(context.Background(), m.httpClientGetPayload.Timeout)
	defer requestCtxCancel()

	// Make a list of relays without SSZ support
	var relaysWithoutSSZ []string
	for _, relay := range originalBid.relays {
		if !relay.SupportsSSZ {
			relaysWithoutSSZ = append(relaysWithoutSSZ, relay.URL.Hostname())
		}
	}

	// Convert the blinded block to JSON if there's a relay that doesn't support SSZ yet
	var signedBlindedBeaconBlockBytesJSON []byte
	if proposerContentType == MediaTypeOctetStream && len(relaysWithoutSSZ) > 0 {
		log.WithField("relaysWithoutSSZ", relaysWithoutSSZ).Info("Converting request from SSZ to JSON for relay(s)")
		signedBlindedBeaconBlockBytesJSON, err = convertSSZToJSON(proposerEthConsensusVersion, signedBlindedBeaconBlockBytes)
		if err != nil {
			log.WithError(errFailedToConvert).Error("failed to convert SSZ to JSON")
			return nil, bidResp{}
		}
	}

	// Only request payloads from relays which provided the bid. This is
	// necessary now because we use the bid to track relay encoding preferences.
	for _, relay := range originalBid.relays {
		go func(relay types.RelayEntry) {
			url := relay.GetURI(params.PathGetPayload)
			log := log.WithField("url", url)
			log.Debug("calling getPayload")

			// If the request fails, try again a few times with 100ms between tries
			resp, err := retry(requestCtx, m.requestMaxRetries, 100*time.Millisecond, func() (*http.Response, error) {
				// If necessary, use the JSON encoded version and the JSON Content-Type header
				requestContentType := parsedProposerContentType
				requestBytes := signedBlindedBeaconBlockBytes
				if parsedProposerContentType == MediaTypeOctetStream && !relay.SupportsSSZ {
					requestBytes = signedBlindedBeaconBlockBytesJSON
					requestContentType = MediaTypeJSON
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
				log.Debug("requesting payload")
				resp, err := m.httpClientGetPayload.Do(req)
				if err != nil {
					log.WithError(err).Warn("error calling getPayload on relay")
					return nil, err
				}

				// Check that the response was successful
				if resp.StatusCode != http.StatusOK {
					err = fmt.Errorf("%w: %d", errHTTPErrorResponse, resp.StatusCode)
					log.WithError(err).Warn("error status code")
					return nil, err
				}

				return resp, nil
			})
			if err != nil {
				log.WithError(err).Warn("failed to get payload from relay after retries")
				return
			}
			defer resp.Body.Close()

			// Get the resp body content
			respBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				log.WithError(err).Warn("error reading response body")
				return
			}

			// Get the response's content type
			respContentType, _, err := mime.ParseMediaType(resp.Header.Get(HeaderContentType))
			if err != nil {
				log.WithError(err).Warn("error parsing response content type")
				respContentType = MediaTypeJSON
			}
			log = log.WithField("respContentType", respContentType)

			// Get the response's eth consensus version
			respEthConsensusVersion := resp.Header.Get(HeaderEthConsensusVersion)
			log = log.WithField("respEthConsensusVersion", respEthConsensusVersion)

			// Decode response
			response := new(builderApi.VersionedSubmitBlindedBlockResponse)
			err = decodeSubmitBlindedBlockResponse(respBytes, respContentType, respEthConsensusVersion, response)
			if err != nil {
				log.WithError(err).Warn("error decoding bid")
				return
			}

			// Check that the payload matches our request
			err = verifyPayload(log, request, response)
			if err != nil {
				log.WithError(err).Warn("error decoding bid")
				return
			}

			// The payload is valid, cancel the request for others
			requestCtxCancel()

			// We have received a payload, cancel other requests
			if received.CompareAndSwap(false, true) {
				resultCh <- response
				log.Info("received payload from relay")
			} else {
				log.Trace("discarding response, already received a correct response")
			}
		}(relay)
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

	// Ensure the blobs bundle field counts are correct
	if len(requestCommitments) != len(responseBlobsBundle.Blobs) ||
		len(requestCommitments) != len(responseBlobsBundle.Commitments) ||
		len(requestCommitments) != len(responseBlobsBundle.Proofs) {
		log.WithFields(logrus.Fields{
			"requestBlobCommitments":  len(requestCommitments),
			"responseBlobs":           len(responseBlobsBundle.Blobs),
			"responseBlobCommitments": len(responseBlobsBundle.Commitments),
			"responseBlobProofs":      len(responseBlobsBundle.Proofs),
		}).Error("different lengths for blobs/commitments/proofs")
		return errInvalidKZGLength
	}

	// Ensure the request and response KZG commitments are the same
	for i, commitment := range requestCommitments {
		if commitment != responseBlobsBundle.Commitments[i] {
			log.WithFields(logrus.Fields{
				"index":                  i,
				"requestBlobCommitment":  commitment.String(),
				"responseBlobCommitment": responseBlobsBundle.Commitments[i].String(),
			}).Error("requestBlobCommitment does not equal responseBlobCommitment")
			return errInvalidKZG
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
		if ethConsensusVersion != "" {
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
			default:
				return errInvalidForkVersion
			}
		} else {
			return types.ErrMissingEthConsensusVersion
		}
	case MediaTypeJSON:
		var err error
		electraBlock := new(eth2ApiV1Electra.SignedBlindedBeaconBlock)
		err = json.Unmarshal(in, electraBlock)
		if err == nil {
			out.Version = spec.DataVersionElectra
			out.Electra = electraBlock
			return nil
		}
		denebBlock := new(eth2ApiV1Deneb.SignedBlindedBeaconBlock)
		err = json.Unmarshal(in, denebBlock)
		if err == nil {
			out.Version = spec.DataVersionDeneb
			out.Deneb = denebBlock
			return nil
		}
		capellaBlock := new(eth2ApiV1Capella.SignedBlindedBeaconBlock)
		err = json.Unmarshal(in, capellaBlock)
		if err == nil {
			out.Version = spec.DataVersionCapella
			out.Capella = capellaBlock
			return nil
		}
		bellatrixBlock := new(eth2ApiV1Bellatrix.SignedBlindedBeaconBlock)
		err = json.Unmarshal(in, bellatrixBlock)
		if err == nil {
			out.Version = spec.DataVersionBellatrix
			out.Bellatrix = bellatrixBlock
			return nil
		}
		return errFailedToDecode
	}
	return types.ErrInvalidContentType
}

// decodeSubmitBlindedBlockResponse will decode the response contents in either JSON or SSZ
func decodeSubmitBlindedBlockResponse(in []byte, contentType, ethConsensusVersion string, out *builderApi.VersionedSubmitBlindedBlockResponse) error {
	switch contentType {
	case MediaTypeOctetStream:
		if ethConsensusVersion != "" {
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
			default:
				return errInvalidForkVersion
			}
		} else {
			return types.ErrMissingEthConsensusVersion
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
