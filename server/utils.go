package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	boostTypes "github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/mev-boost/config"
	"github.com/flashbots/mev-boost/config/rcp"
)

var (
	errHTTPErrorResponse  = errors.New("HTTP error response")
	errInvalidForkVersion = errors.New("invalid fork version")
	errInvalidTransaction = errors.New("invalid transaction")
)

// UserAgent is a custom string type to avoid confusing url + userAgent parameters in SendHTTPRequest
type UserAgent string

// BlockHashHex is a hex-string representation of a block hash
type BlockHashHex string

// SendHTTPRequest - prepare and send HTTP request, marshaling the payload if any, and decoding the response if dst is set
func SendHTTPRequest(ctx context.Context, client http.Client, method, url string, userAgent UserAgent, payload, dst any) (code int, err error) {
	var req *http.Request

	if payload == nil {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	} else {
		payloadBytes, err2 := json.Marshal(payload)
		if err2 != nil {
			return 0, fmt.Errorf("could not marshal request: %w", err2)
		}
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payloadBytes))

		// Set headers
		req.Header.Add("Content-Type", "application/json")
	}
	if err != nil {
		return 0, fmt.Errorf("could not prepare request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", strings.TrimSpace(fmt.Sprintf("mev-boost/%s %s", config.Version, userAgent)))

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
func ComputeDomain(domainType boostTypes.DomainType, forkVersionHex, genesisValidatorsRootHex string) (domain boostTypes.Domain, err error) {
	genesisValidatorsRoot := boostTypes.Root(common.HexToHash(genesisValidatorsRootHex))
	forkVersionBytes, err := hexutil.Decode(forkVersionHex)
	if err != nil || len(forkVersionBytes) != 4 {
		return domain, errInvalidForkVersion
	}
	var forkVersion [4]byte
	copy(forkVersion[:], forkVersionBytes[:4])
	return boostTypes.ComputeDomain(domainType, forkVersion, genesisValidatorsRoot), nil
}

// DecodeJSON reads JSON from io.Reader and decodes it into a struct
func DecodeJSON(r io.Reader, dst any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}
	return nil
}

// GetURI returns the full request URI with scheme, host, path and args.
func GetURI(url *url.URL, path string) string {
	u2 := *url
	u2.User = nil
	u2.Path = path
	return u2.String()
}

// bidResp are entries in the bids cache
type bidResp struct {
	t         time.Time
	response  boostTypes.GetHeaderResponse
	blockHash string
	relays    []RelayEntry
}

// bidRespKey is used as key for the bids cache
type bidRespKey struct {
	slot      uint64
	blockHash string
}

func httpClientDisallowRedirects(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}

func weiBigIntToEthBigFloat(wei *big.Int) (ethValue *big.Float) {
	// wei / 10^18
	fbalance := new(big.Float)
	fbalance.SetString(wei.String())
	ethValue = new(big.Float).Quo(fbalance, big.NewFloat(1e18))
	return
}

func ComputeBlockHash(payload *capella.ExecutionPayload) (phase0.Hash32, error) {
	header, err := executionPayloadToBlockHeader(payload)
	if err != nil {
		return phase0.Hash32{}, err
	}
	return phase0.Hash32(header.Hash()), nil
}

func executionPayloadToBlockHeader(payload *capella.ExecutionPayload) (*types.Header, error) {
	transactionData := make([]*types.Transaction, len(payload.Transactions))
	for i, encTx := range payload.Transactions {
		var tx types.Transaction

		if err := tx.UnmarshalBinary(encTx); err != nil {
			return nil, errInvalidTransaction
		}
		transactionData[i] = &tx
	}

	// base fee per gas is stored little-endian but we need it
	// big-endian for big.Int.
	var baseFeePerGasBytes [32]byte
	for i := 0; i < 32; i++ {
		baseFeePerGasBytes[i] = payload.BaseFeePerGas[32-1-i]
	}
	baseFeePerGas := new(big.Int).SetBytes(baseFeePerGasBytes[:])

	return &types.Header{
		ParentHash:  common.Hash(payload.ParentHash),
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    common.Address(payload.FeeRecipient),
		Root:        common.Hash(payload.StateRoot),
		TxHash:      types.DeriveSha(types.Transactions(transactionData), trie.NewStackTrie(nil)),
		ReceiptHash: common.Hash(payload.ReceiptsRoot),
		Bloom:       types.Bloom(payload.LogsBloom),
		Difficulty:  common.Big0,
		Number:      new(big.Int).SetUint64(payload.BlockNumber),
		GasLimit:    payload.GasLimit,
		GasUsed:     payload.GasUsed,
		Time:        payload.Timestamp,
		Extra:       payload.ExtraData,
		MixDigest:   common.Hash(payload.PrevRandao),
		BaseFee:     baseFeePerGas,
	}, nil
}

func RelayEntriesToRCMRelayEntries(relays []RelayEntry) []rcp.RelayEntry {
	relayEntries := make([]rcp.RelayEntry, len(relays))
	for i, relay := range relays {
		relayEntries[i] = relay
	}

	return relayEntries
}

func RCMRelayEntriesToRelayEntries(relays []rcp.RelayEntry) []RelayEntry {
	relayEntries := make([]RelayEntry, len(relays))
	for i, relay := range relays {
		relayEntries[i] = RelayEntry{
			PublicKey: relay.PubKey(),
			URL:       relay.RelayURL(),
		}
	}

	return relayEntries
}

func RCMRelayEntryToRelayEntry(relay rcp.RelayEntry) RelayEntry {
	return RelayEntry{
		PublicKey: relay.PubKey(),
		URL:       relay.RelayURL(),
	}
}
