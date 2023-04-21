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

	builderApi "github.com/attestantio/go-builder-client/api"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/flashbots/go-boost-utils/ssz"
	"github.com/flashbots/mev-boost/config"
	"github.com/sirupsen/logrus"
)

var (
	errHTTPErrorResponse  = errors.New("HTTP error response")
	errInvalidForkVersion = errors.New("invalid fork version")
	errInvalidTransaction = errors.New("invalid transaction")
	errMaxRetriesExceeded = errors.New("max retries exceeded")
	errInvalidPayload     = errors.New("invalid payload")
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

// SendHTTPRequestWithRetries - prepare and send HTTP request, retrying the request if within the client timeout
func SendHTTPRequestWithRetries(ctx context.Context, client http.Client, method, url string, userAgent UserAgent, payload, dst any, maxRetries int, log *logrus.Entry) (code int, err error) {
	var requestCtx context.Context
	var cancel context.CancelFunc
	if client.Timeout > 0 {
		// Create a context with a timeout as configured in the http client
		requestCtx, cancel = context.WithTimeout(context.Background(), client.Timeout)
	} else {
		requestCtx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	attempts := 0
	for {
		attempts++
		if requestCtx.Err() != nil {
			return 0, fmt.Errorf("request context error after %d attempts: %w", attempts, requestCtx.Err())
		}
		if attempts > maxRetries {
			return 0, errMaxRetriesExceeded
		}

		code, err = SendHTTPRequest(ctx, client, method, url, userAgent, payload, dst)
		if err != nil {
			log.WithError(err).Warn("error making request to relay, retrying")
			time.Sleep(100 * time.Millisecond) // note: this timeout is only applied between retries, it does not delay the initial request!
			continue
		}
		return code, nil
	}
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
	response  GetHeaderResponse
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

func ComputeBlockHash(payload *builderApi.VersionedExecutionPayload) (phase0.Hash32, error) {
	header, err := executionPayloadToBlockHeader(payload)
	if err != nil {
		return phase0.Hash32{}, err
	}
	return phase0.Hash32(header.Hash()), nil
}

func bigIntBaseFeePerGas(baseFeePerGas [32]byte) *big.Int {
	// baseFeePerGas is little-endian but we need big-endian for big.Int
	var baseFeePerGasBytes [32]byte
	for i := 0; i < 32; i++ {
		baseFeePerGasBytes[i] = baseFeePerGas[32-1-i]
	}
	return new(big.Int).SetBytes(baseFeePerGasBytes[:])
}

func hashTransactions(transactions []bellatrix.Transaction) (common.Hash, error) {
	transactionData := make([]*types.Transaction, len(transactions))
	for i, encTx := range transactions {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(encTx); err != nil {
			return common.Hash{}, errInvalidTransaction
		}
		transactionData[i] = &tx
	}
	return types.DeriveSha(types.Transactions(transactionData), trie.NewStackTrie(nil)), nil
}

func hashWithdrawals(withdrawals []*capella.Withdrawal) (common.Hash, error) {
	withdrawalData := make([]*types.Withdrawal, len(withdrawals))
	for i, w := range withdrawals {
		withdrawalData[i] = &types.Withdrawal{
			Index:     uint64(w.Index),
			Validator: uint64(w.ValidatorIndex),
			Address:   common.Address(w.Address),
			Amount:    uint64(w.Amount),
		}
	}
	return types.DeriveSha(types.Withdrawals(withdrawalData), trie.NewStackTrie(nil)), nil
}

func executionPayloadToBlockHeader(payload *builderApi.VersionedExecutionPayload) (*types.Header, error) {
	switch payload.Version {
	case spec.DataVersionBellatrix:
		baseFeePerGas := bigIntBaseFeePerGas(payload.Bellatrix.BaseFeePerGas)
		transactionsHash, err := hashTransactions(payload.Bellatrix.Transactions)
		if err != nil {
			return nil, err
		}
		return &types.Header{
			ParentHash:  common.Hash(payload.Bellatrix.ParentHash),
			UncleHash:   types.EmptyUncleHash,
			Coinbase:    common.Address(payload.Bellatrix.FeeRecipient),
			Root:        payload.Bellatrix.StateRoot,
			TxHash:      transactionsHash,
			ReceiptHash: payload.Bellatrix.ReceiptsRoot,
			Bloom:       payload.Bellatrix.LogsBloom,
			Difficulty:  common.Big0,
			Number:      new(big.Int).SetUint64(payload.Bellatrix.BlockNumber),
			GasLimit:    payload.Bellatrix.GasLimit,
			GasUsed:     payload.Bellatrix.GasUsed,
			Time:        payload.Bellatrix.Timestamp,
			Extra:       payload.Bellatrix.ExtraData,
			MixDigest:   payload.Bellatrix.PrevRandao,
			BaseFee:     baseFeePerGas,
		}, nil
	case spec.DataVersionCapella:
		baseFeePerGas := bigIntBaseFeePerGas(payload.Capella.BaseFeePerGas)
		transactionsHash, err := hashTransactions(payload.Capella.Transactions)
		if err != nil {
			return nil, err
		}
		withdrawalsHash, err := hashWithdrawals(payload.Capella.Withdrawals)
		if err != nil {
			return nil, err
		}
		return &types.Header{
			ParentHash:      common.Hash(payload.Capella.ParentHash),
			UncleHash:       types.EmptyUncleHash,
			Coinbase:        common.Address(payload.Capella.FeeRecipient),
			Root:            payload.Capella.StateRoot,
			TxHash:          transactionsHash,
			ReceiptHash:     payload.Capella.ReceiptsRoot,
			Bloom:           payload.Capella.LogsBloom,
			Difficulty:      common.Big0,
			Number:          new(big.Int).SetUint64(payload.Capella.BlockNumber),
			GasLimit:        payload.Capella.GasLimit,
			GasUsed:         payload.Capella.GasUsed,
			Time:            payload.Capella.Timestamp,
			Extra:           payload.Capella.ExtraData,
			MixDigest:       payload.Capella.PrevRandao,
			BaseFee:         baseFeePerGas,
			WithdrawalsHash: &withdrawalsHash,
		}, nil
	default:
		return &types.Header{}, errInvalidPayload
	}
}
