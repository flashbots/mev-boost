package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"sync"
	"time"

	builderApiBellatrix "github.com/attestantio/go-builder-client/api/bellatrix"
	builderApiCapella "github.com/attestantio/go-builder-client/api/capella"
	builderApiDeneb "github.com/attestantio/go-builder-client/api/deneb"
	builderApiElectra "github.com/attestantio/go-builder-client/api/electra"
	builderSpec "github.com/attestantio/go-builder-client/spec"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/flashbots/mev-boost/config"
	"github.com/flashbots/mev-boost/server/types"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// getHeader requests a bid from each relay and returns the most profitable one
func (m *BoostService) getHeader(log *logrus.Entry, slot phase0.Slot, pubkey, parentHashHex string, ua UserAgent, proposerAcceptContentTypes string) (bidResp, error) {
	// Ensure arguments are valid
	if len(pubkey) != 98 {
		return bidResp{}, errInvalidPubkey
	}
	if len(parentHashHex) != 66 {
		return bidResp{}, errInvalidHash
	}

	// Make sure we have a uid for this slot
	m.slotUIDLock.Lock()
	if m.slotUID.slot < slot {
		m.slotUID.slot = slot
		m.slotUID.uid = uuid.New()
	}
	slotUID := m.slotUID.uid
	m.slotUIDLock.Unlock()
	log = log.WithField("slotUID", slotUID)

	// Compute these once, instead of for each relay
	userAgent := wrapUserAgent(ua)
	startTime := fmt.Sprintf("%d", time.Now().UTC().UnixMilli())

	// Log how late into the slot the request starts
	slotStartTimestamp := m.genesisTime + uint64(slot)*config.SlotTimeSec
	msIntoSlot := uint64(time.Now().UTC().UnixMilli()) - slotStartTimestamp*1000
	log.WithFields(logrus.Fields{
		"genesisTime": m.genesisTime,
		"slotTimeSec": config.SlotTimeSec,
		"msIntoSlot":  msIntoSlot,
	}).Infof("getHeader request start - %d milliseconds into slot %d", msIntoSlot, slot)

	var (
		mu sync.Mutex
		wg sync.WaitGroup

		// The final response, containing the highest bid (if any)
		result = bidResp{}

		// Relays that sent the bid for a specific blockHash
		relays = make(map[BlockHashHex][]types.RelayEntry)
	)

	// Request a bid from each relay
	for _, relay := range m.relays {
		wg.Add(1)
		go func(relay types.RelayEntry) {
			defer wg.Done()

			// Build the request URL
			url := relay.GetURI(fmt.Sprintf("/eth/v1/builder/header/%d/%s/%s", slot, parentHashHex, pubkey))
			log := log.WithField("url", url)

			// Make a new request
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
			if err != nil {
				log.WithError(err).Warn("error creating new request")
				return
			}

			// Add header fields to this request
			req.Header.Set(HeaderAccept, proposerAcceptContentTypes)
			req.Header.Set(HeaderKeySlotUID, slotUID.String())
			req.Header.Set(HeaderStartTimeUnixMS, startTime)
			req.Header.Set(HeaderUserAgent, userAgent)
			req.Header.Set(HeaderDateMilliseconds, startTime)

			// Send the request
			log.Debug("requesting header")
			resp, err := m.httpClientGetHeader.Do(req)
			if err != nil {
				log.WithError(err).Warn("error calling getHeader on relay")
				return
			}
			defer resp.Body.Close()

			// Check if no header is available
			if resp.StatusCode == http.StatusNoContent {
				log.Debug("no-content response")
				return
			}

			// Check that the response was successful
			if resp.StatusCode != http.StatusOK {
				err = fmt.Errorf("%w: %d", errHTTPErrorResponse, resp.StatusCode)
				log.WithError(err).Warn("error status code")
				return
			}

			// Get the resp body content
			respBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				log.WithError(err).Warn("error reading response body")
				return
			}

			// Get the response's content type, default to JSON
			respContentType, _, err := mime.ParseMediaType(resp.Header.Get(HeaderContentType))
			if err != nil {
				log.WithError(err).Warn("error parsing response content type")
				respContentType = MediaTypeJSON
			}
			log = log.WithField("respContentType", respContentType)

			// Get the optional version, used with SSZ decoding
			respEthConsensusVersion := resp.Header.Get(HeaderEthConsensusVersion)
			log = log.WithField("respEthConsensusVersion", respEthConsensusVersion)

			// Decode bid
			bid := new(builderSpec.VersionedSignedBuilderBid)
			err = decodeBid(respBytes, respContentType, respEthConsensusVersion, bid)
			if err != nil {
				log.WithError(err).Warn("error decoding bid")
				return
			}

			// Skip if bid is empty
			if bid.IsEmpty() {
				log.Debug("skipping empty bid")
				return
			}

			// Getting the bid info will check if there are missing fields in the response
			bidInfo, err := parseBidInfo(bid)
			if err != nil {
				log.WithError(err).Warn("error parsing bid info")
				return
			}

			// Ignore bids with an empty block
			if bidInfo.blockHash == nilHash {
				log.Warn("relay responded with empty block hash")
				return
			}

			// Add some info about the bid to the logger
			valueEth := weiBigIntToEthBigFloat(bidInfo.value.ToBig())
			log = log.WithFields(logrus.Fields{
				"blockNumber": bidInfo.blockNumber,
				"blockHash":   bidInfo.blockHash.String(),
				"txRoot":      bidInfo.txRoot.String(),
				"value":       valueEth.Text('f', 18),
			})

			// Ensure the bid uses the correct public key
			if relay.PublicKey.String() != bidInfo.pubkey.String() {
				log.Errorf("bid pubkey mismatch. expected: %s - got: %s", relay.PublicKey.String(), bidInfo.pubkey.String())
				return
			}

			// Verify the relay signature in the relay response
			if !config.SkipRelaySignatureCheck {
				ok, err := checkRelaySignature(bid, m.builderSigningDomain, relay.PublicKey)
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

			// Ignore bids with 0 value
			isZeroValue := bidInfo.value.IsZero()
			isEmptyListTxRoot := bidInfo.txRoot.String() == "0x7ffe241ea60187fdb0187bfa22de35d1f9bed7ab061d9401fd47e34a54fbede1"
			if isZeroValue || isEmptyListTxRoot {
				log.Warn("ignoring bid with 0 value")
				return
			}

			log.Debug("bid received")

			// Skip if value is lower than the minimum bid
			if bidInfo.value.CmpBig(m.relayMinBid.BigInt()) == -1 {
				log.Debug("ignoring bid below min-bid value")
				return
			}

			mu.Lock()
			defer mu.Unlock()

			// Create a copy of the relay instance with its encoding preference. If we request SSZ and the relay
			// responds with JSON, we know that it does not support SSZ yet. This preference will be used in getPayload,
			// because we must encode the blinded block in the request in such a way that the relay can decode it.
			relayWithEncodingPreference := relay.Copy()
			relayWithEncodingPreference.SupportsSSZ = respContentType == MediaTypeOctetStream

			// Remember which relays delivered which bids (multiple relays might deliver the top bid)
			relays[BlockHashHex(bidInfo.blockHash.String())] = append(relays[BlockHashHex(bidInfo.blockHash.String())], relayWithEncodingPreference)

			// Compare the bid with already known top bid (if any)
			if !result.response.IsEmpty() {
				valueDiff := bidInfo.value.Cmp(result.bidInfo.value)
				if valueDiff == -1 {
					// The current bid is less profitable than already known one
					log.Debug("ignoring less profitable bid")
					return
				} else if valueDiff == 0 {
					// The current bid is equally profitable as already known one
					// Use hash as tiebreaker
					previousBidBlockHash := result.bidInfo.blockHash
					if bidInfo.blockHash.String() >= previousBidBlockHash.String() {
						log.Debug("equally profitable bid lost tiebreaker")
						return
					}
				}
			}

			// Use this relay's response as mev-boost response because it's most profitable
			log.Debug("new best bid")
			result.response = *bid
			result.bidInfo = bidInfo
			result.t = time.Now()
		}(relay)
	}
	wg.Wait()

	// Set the winning relays before returning
	result.relays = relays[BlockHashHex(result.bidInfo.blockHash.String())]
	return result, nil
}

// decodeBid decodes a bid by SSZ or JSON, depending on the provided respContentType
func decodeBid(respBytes []byte, respContentType, ethConsensusVersion string, bid *builderSpec.VersionedSignedBuilderBid) error {
	switch respContentType {
	case MediaTypeOctetStream:
		if ethConsensusVersion != "" {
			// Do SSZ decoding
			switch ethConsensusVersion {
			case EthConsensusVersionBellatrix:
				bid.Version = spec.DataVersionBellatrix
				bid.Bellatrix = new(builderApiBellatrix.SignedBuilderBid)
				return bid.Bellatrix.UnmarshalSSZ(respBytes)
			case EthConsensusVersionCapella:
				bid.Version = spec.DataVersionCapella
				bid.Capella = new(builderApiCapella.SignedBuilderBid)
				return bid.Capella.UnmarshalSSZ(respBytes)
			case EthConsensusVersionDeneb:
				bid.Version = spec.DataVersionDeneb
				bid.Deneb = new(builderApiDeneb.SignedBuilderBid)
				return bid.Deneb.UnmarshalSSZ(respBytes)
			case EthConsensusVersionElectra:
				bid.Version = spec.DataVersionElectra
				bid.Electra = new(builderApiElectra.SignedBuilderBid)
				return bid.Electra.UnmarshalSSZ(respBytes)
			default:
				return errInvalidForkVersion
			}
		} else {
			return types.ErrMissingEthConsensusVersion
		}
	case MediaTypeJSON:
		// Do JSON decoding
		return json.Unmarshal(respBytes, bid)
	}
	return types.ErrInvalidContentType
}

// respondGetHeaderJSON responds to the proposer in JSON
func (m *BoostService) respondGetHeaderJSON(w http.ResponseWriter, result *bidResp) {
	w.Header().Set(HeaderContentType, MediaTypeJSON)
	w.WriteHeader(http.StatusOK)

	// Serialize and write the data
	if err := json.NewEncoder(w).Encode(&result.response); err != nil {
		m.log.WithField("response", result.response).WithError(err).Error("could not write OK response")
		http.Error(w, "", http.StatusInternalServerError)
	}
}

// respondGetHeaderSSZ responds to the proposer in SSZ
func (m *BoostService) respondGetHeaderSSZ(w http.ResponseWriter, result *bidResp) {
	// Serialize the response
	var err error
	var sszData []byte
	switch result.response.Version {
	case spec.DataVersionBellatrix:
		w.Header().Set(HeaderEthConsensusVersion, EthConsensusVersionBellatrix)
		sszData, err = result.response.Bellatrix.MarshalSSZ()
	case spec.DataVersionCapella:
		w.Header().Set(HeaderEthConsensusVersion, EthConsensusVersionCapella)
		sszData, err = result.response.Capella.MarshalSSZ()
	case spec.DataVersionDeneb:
		w.Header().Set(HeaderEthConsensusVersion, EthConsensusVersionDeneb)
		sszData, err = result.response.Deneb.MarshalSSZ()
	case spec.DataVersionElectra:
		w.Header().Set(HeaderEthConsensusVersion, EthConsensusVersionElectra)
		sszData, err = result.response.Electra.MarshalSSZ()
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
