package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"encoding/hex"

	"github.com/OffchainLabs/prysm/v6/runtime/interop"
	builderApi "github.com/attestantio/go-builder-client/api"
	builderSpec "github.com/attestantio/go-builder-client/spec"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/go-boost-utils/bls"
	"github.com/flashbots/go-boost-utils/ssz"
	"github.com/flashbots/mev-boost/config"
	"github.com/flashbots/mev-boost/server/types"
	"github.com/holiman/uint256"
)

var (
	errHTTPErrorResponse  = errors.New("HTTP error response")
	errInvalidForkVersion = errors.New("invalid fork version")
)

// UserAgent is a custom string type to avoid confusing url + userAgent parameters in SendHTTPRequest
type UserAgent string

// BlockHashHex is a hex-string representation of a block hash
type BlockHashHex string

// SendHTTPRequest - prepare and send HTTP request, marshaling the payload if any, and decoding the response if dst is set
func SendHTTPRequest(ctx context.Context, client http.Client, method, url string, userAgent UserAgent, headers map[string]string, payload, dst any) (code int, err error) {
	var req *http.Request

	if payload == nil {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return 0, fmt.Errorf("could not prepare request: %w", err)
		}
	} else {
		payloadBytes, err2 := json.Marshal(payload)
		if err2 != nil {
			return 0, fmt.Errorf("could not marshal request: %w", err2)
		}
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payloadBytes))
		if err != nil {
			return 0, fmt.Errorf("could not prepare request: %w", err)
		}
		// Set Content-Type header
		req.Header.Add("Content-Type", "application/json")
	}

	// Set User-Agent header
	req.Header.Set("User-Agent", strings.TrimSpace(fmt.Sprintf("mev-boost/%s %s", config.Version, userAgent)))

	// Set other headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return resp.StatusCode, nil
	}

	if resp.StatusCode > 299 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return resp.StatusCode, fmt.Errorf("could not read error response body for status code %d: %w", resp.StatusCode, err)
		}
		return resp.StatusCode, fmt.Errorf("%w: %d / %s", errHTTPErrorResponse, resp.StatusCode, string(bodyBytes))
	}

	if dst != nil {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return resp.StatusCode, fmt.Errorf("could not read response body: %w", err)
		}

		if err := json.Unmarshal(bodyBytes, dst); err != nil {
			return resp.StatusCode, fmt.Errorf("could not unmarshal response %s: %w", string(bodyBytes), err)
		}
	}

	return resp.StatusCode, nil
}

// ComputeDomain computes the signing domain
func ComputeDomain(domainType phase0.DomainType, forkVersionHex, genesisValidatorsRootHex string) (domain phase0.Domain, err error) {
	genesisValidatorsRoot := phase0.Root(common.HexToHash(genesisValidatorsRootHex))
	forkVersionBytes, err := hexutil.Decode(forkVersionHex)
	if err != nil || len(forkVersionBytes) != 4 {
		return domain, errInvalidForkVersion
	}
	var forkVersion [4]byte
	copy(forkVersion[:], forkVersionBytes[:4])
	return ssz.ComputeDomain(domainType, forkVersion, genesisValidatorsRoot), nil
}

// DecodeJSON reads JSON from io.Reader and decodes it into a struct
func DecodeJSON(r io.Reader, dst any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

// bidResp are entries in the bids cache
type bidResp struct {
	t        time.Time
	response builderSpec.VersionedSignedBuilderBid
	bidInfo  bidInfo
	relays   []types.RelayEntry
}

// bidInfo is used to store bid response fields for logging and validation
type bidInfo struct {
	blockHash   phase0.Hash32
	parentHash  phase0.Hash32
	pubkey      phase0.BLSPubKey
	blockNumber uint64
	txRoot      phase0.Root
	value       *uint256.Int
}

func httpClientDisallowRedirects(_ *http.Request, _ []*http.Request) error {
	return http.ErrUseLastResponse
}

func weiBigIntToEthBigFloat(wei *big.Int) (ethValue *big.Float) {
	// wei / 10^18
	fbalance := new(big.Float)
	fbalance.SetString(wei.String())
	ethValue = new(big.Float).Quo(fbalance, big.NewFloat(1e18))
	return
}

func parseBidInfo(bid *builderSpec.VersionedSignedBuilderBid) (bidInfo, error) {
	blockHash, err := bid.BlockHash()
	if err != nil {
		return bidInfo{}, err
	}
	parentHash, err := bid.ParentHash()
	if err != nil {
		return bidInfo{}, err
	}
	pubkey, err := bid.Builder()
	if err != nil {
		return bidInfo{}, err
	}
	blockNumber, err := bid.BlockNumber()
	if err != nil {
		return bidInfo{}, err
	}
	txRoot, err := bid.TransactionsRoot()
	if err != nil {
		return bidInfo{}, err
	}
	value, err := bid.Value()
	if err != nil {
		return bidInfo{}, err
	}
	return bidInfo{
		blockHash:   blockHash,
		parentHash:  parentHash,
		pubkey:      pubkey,
		blockNumber: blockNumber,
		txRoot:      txRoot,
		value:       value,
	}, nil
}

func checkRelaySignature(bid *builderSpec.VersionedSignedBuilderBid, domain phase0.Domain, pubKey phase0.BLSPubKey) (bool, error) {
	root, err := bid.MessageHashTreeRoot()
	if err != nil {
		return false, err
	}
	sig, err := bid.Signature()
	if err != nil {
		return false, err
	}
	signingData := phase0.SigningData{ObjectRoot: root, Domain: domain}
	msg, err := signingData.HashTreeRoot()
	if err != nil {
		return false, err
	}

	return bls.VerifySignatureBytes(msg[:], sig[:], pubKey[:])
}

func getPayloadResponseIsEmpty(payload *builderApi.VersionedSubmitBlindedBlockResponse) bool {
	switch payload.Version {
	case spec.DataVersionBellatrix:
		if payload.Bellatrix == nil || payload.Bellatrix.BlockHash == nilHash {
			return true
		}
	case spec.DataVersionCapella:
		if payload.Capella == nil || payload.Capella.BlockHash == nilHash {
			return true
		}
	case spec.DataVersionDeneb:
		if payload.Deneb == nil || payload.Deneb.ExecutionPayload == nil ||
			payload.Deneb.ExecutionPayload.BlockHash == nilHash ||
			payload.Deneb.BlobsBundle == nil {
			return true
		}
	case spec.DataVersionElectra:
		if payload.Electra == nil || payload.Electra.ExecutionPayload == nil ||
			payload.Electra.ExecutionPayload.BlockHash == nilHash ||
			payload.Electra.BlobsBundle == nil {
			return true
		}
	case spec.DataVersionUnknown, spec.DataVersionPhase0, spec.DataVersionAltair:
		return true
	}
	return false
}

func wrapUserAgent(ua UserAgent) string {
	return strings.TrimSpace(fmt.Sprintf("mev-boost/%s %s", config.Version, ua))
}

// ValidatorKeyMap stores public key -> private key mapping
type ValidatorKeyMap struct {
	keys map[string][]byte // public key hex -> private key bytes
}

// NewValidatorKeyMap creates a new validator key map
func NewValidatorKeyMap() *ValidatorKeyMap {
	return &ValidatorKeyMap{
		keys: make(map[string][]byte),
	}
}

// GetSigningKey returns the private key for a given public key
func (vkm *ValidatorKeyMap) GetSigningKey(publicKeyHex string) ([]byte, bool) {
	privateKey, exists := vkm.keys[publicKeyHex]
	return privateKey, exists
}

// GetSigningKeyHex returns the private key as hex string for a given public key
func (vkm *ValidatorKeyMap) GetSigningKeyHex(publicKeyHex string) (string, bool) {
	privateKey, exists := vkm.keys[publicKeyHex]
	if !exists {
		return "", false
	}
	return hex.EncodeToString(privateKey), true
}

// GetAllPublicKeys returns all public keys in the map
func (vkm *ValidatorKeyMap) GetAllPublicKeys() []string {
	keys := make([]string, 0, len(vkm.keys))
	for pubKey := range vkm.keys {
		keys = append(keys, pubKey)
	}
	return keys
}

// Count returns the total number of validators
func (vkm *ValidatorKeyMap) Count() int {
	return len(vkm.keys)
}

// Global validator key map for integration tests
var globalValidatorKeyMap *ValidatorKeyMap

// initValidators initializes validator keys for integration testing
func initValidators() {
	// Create validator key map
	globalValidatorKeyMap = NewValidatorKeyMap()

	// Generate the same 100 validator keys as builder playground
	privKeys, pubKeys, err := interop.DeterministicallyGenerateKeys(0, 100)
	if err != nil {
		log.Printf("Warning: Failed to generate keys: %v", err)
		return
	}

	// Store keys in the map
	for i, privKey := range privKeys {
		pubKey := pubKeys[i]

		// Convert public key to hex string
		pubKeyHex := "0x" + hex.EncodeToString(pubKey.Marshal())

		// Store private key in the map
		globalValidatorKeyMap.keys[pubKeyHex] = privKey.Marshal()

		if i < 5 { // Only log first 5 to avoid spam
			log.Printf("Stored validator %d: %s\n", i, pubKeyHex)
		}
	}
	log.Printf("Initialized %d validator keys", globalValidatorKeyMap.Count())
}

// GetValidatorKeyMap returns the global validator key map
func GetValidatorKeyMap() *ValidatorKeyMap {
	return globalValidatorKeyMap
}
