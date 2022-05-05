package types

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// NilHash represents an empty hash
var NilHash = common.Hash{}

// BlindedBeaconBlockV1 spec: https://github.com/ethereum/execution-apis/pull/209/files#diff-20ad1b5e044517450181107c321127eff3a6beeb9deaee4906c9c90cc778daf9R28
type BlindedBeaconBlockV1 struct {
	Slot          string                   `json:"slot"`
	ProposerIndex string                   `json:"proposerIndex"`
	ParentRoot    string                   `json:"parentRoot"`
	StateRoot     string                   `json:"stateRoot"`
	Body          BlindedBeaconBlockBodyV1 `json:"body"`
}

// BlindedBeaconBlockBodyV1 spec: https://github.com/ethereum/execution-apis/pull/209/files#diff-20ad1b5e044517450181107c321127eff3a6beeb9deaee4906c9c90cc778daf9R35
type BlindedBeaconBlockBodyV1 struct {
	RandaoReveal hexutil.Bytes   `json:"randaoReveal"`
	Eth1Data     json.RawMessage `json:"eth1Data"`
	Graffiti     hexutil.Bytes   `json:"graffiti"` // Bytes32  # Arbitrary data

	ProposerSlashings json.RawMessage `json:"proposerSlashings"` // List[ProposerSlashing, MAX_PROPOSER_SLASHINGS]
	AttesterSlashings json.RawMessage `json:"attesterSlashings"` // List[AttesterSlashing, MAX_ATTESTER_SLASHINGS]
	Attestations      json.RawMessage `json:"attestations"`      // List[Attestation, MAX_ATTESTATIONS]
	Deposits          json.RawMessage `json:"deposits"`          // List[Deposit, MAX_DEPOSITS]
	VoluntaryExits    json.RawMessage `json:"voluntaryExits"`    // List[SignedVoluntaryExit, MAX_VOLUNTARY_EXITS]

	SyncAggregate          json.RawMessage          `json:"syncAggregate"` // `object`, [`SyncAggregateV1`](#syncaggregatev1)
	ExecutionPayloadHeader ExecutionPayloadHeaderV1 `json:"executionPayloadHeader"`
}

//go:generate go run github.com/fjl/gencodec -type ExecutionPayloadHeaderV1 -field-override executionPayloadMarshallingOverrides -out gen_executionpayloadheader.go
// ExecutionPayloadHeaderV1 spec https://github.com/ethereum/consensus-specs/blob/dev/specs/bellatrix/beacon-chain.md#executionpayloadheader
type ExecutionPayloadHeaderV1 struct {
	ParentHash       common.Hash    `json:"parentHash" gencodec:"required"`
	FeeRecipient     common.Address `json:"feeRecipient" gencodec:"required"`
	StateRoot        common.Hash    `json:"stateRoot" gencodec:"required"`
	ReceiptsRoot     common.Hash    `json:"receiptsRoot" gencodec:"required"`
	LogsBloom        []byte         `json:"logsBloom" gencodec:"required"`
	PrevRandao       common.Hash    `json:"prevRandao" gencodec:"required"`
	BlockNumber      uint64         `json:"blockNumber" gencodec:"required"`
	GasLimit         uint64         `json:"gasLimit" gencodec:"required"`
	GasUsed          uint64         `json:"gasUsed" gencodec:"required"`
	Timestamp        uint64         `json:"timestamp" gencodec:"required"`
	ExtraData        []byte         `json:"extraData" gencodec:"required"`
	BaseFeePerGas    *big.Int       `json:"baseFeePerGas" gencodec:"required"`
	BlockHash        common.Hash    `json:"blockHash" gencodec:"required"`
	TransactionsRoot common.Hash    `json:"transactionsRoot" gencodec:"required"`
}

//go:generate go run github.com/fjl/gencodec -type ExecutionPayloadV1 -field-override executionPayloadMarshallingOverrides -out gen_executionpayload.go
// ExecutionPayloadV1 as defined in https://github.com/ethereum/execution-apis/blob/v1.0.0-alpha.8/src/engine/specification.md#executionpayloadv1
type ExecutionPayloadV1 struct {
	ParentHash    common.Hash    `json:"parentHash" gencodec:"required"`
	FeeRecipient  common.Address `json:"feeRecipient" gencodec:"required"`
	StateRoot     common.Hash    `json:"stateRoot" gencodec:"required"`
	ReceiptsRoot  common.Hash    `json:"receiptsRoot" gencodec:"required"`
	LogsBloom     []byte         `json:"logsBloom" gencodec:"required"`
	PrevRandao    common.Hash    `json:"prevRandao" gencodec:"required"`
	BlockNumber   uint64         `json:"blockNumber" gencodec:"required"`
	GasLimit      uint64         `json:"gasLimit" gencodec:"required"`
	GasUsed       uint64         `json:"gasUsed" gencodec:"required"`
	Timestamp     uint64         `json:"timestamp" gencodec:"required"`
	ExtraData     []byte         `json:"extraData" gencodec:"required"`
	BaseFeePerGas *big.Int       `json:"baseFeePerGas" gencodec:"required"`
	BlockHash     common.Hash    `json:"blockHash" gencodec:"required"`
	Transactions  *[]string      `json:"transactions" gencodec:"required"`
}

// JSON type overrides for executableData.
type executionPayloadMarshallingOverrides struct {
	BlockNumber   hexutil.Uint64
	GasLimit      hexutil.Uint64
	GasUsed       hexutil.Uint64
	Timestamp     hexutil.Uint64
	BaseFeePerGas *hexutil.Big
	ExtraData     hexutil.Bytes
	LogsBloom     hexutil.Bytes
}

// RegisterValidatorRequestMessage as defined in https://github.com/flashbots/mev-boost/blob/main/docs/specification.md#request
type RegisterValidatorRequestMessage struct {
	FeeRecipient common.Address `json:"fee_recipient"`
	GasLimit     string         `json:"gas_limit"`
	Timestamp    string         `json:"timestamp"`
	Pubkey       hexutil.Bytes  `json:"pubkey"`
}

// RegisterValidatorRequest as defined in XXX
type RegisterValidatorRequest struct {
	Message   RegisterValidatorRequestMessage `json:"message"`
	Signature hexutil.Bytes                   `json:"signature"`
}

// // RegisterValidatorResponse todo
// type RegisterValidatorResponse struct {
// 	Status string `json:"status"`
// }

// GetHeaderResponseMessage as defined in https://github.com/flashbots/mev-boost/blob/main/docs/specification.md#response-1
type GetHeaderResponseMessage struct {
	Header ExecutionPayloadHeaderV1 `json:"header"`
	Value  *hexutil.Big             `json:"value"`
	Pubkey hexutil.Bytes            `json:"pubkey"`
}

// GetHeaderResponse as defined in https://github.com/flashbots/mev-boost/blob/main/docs/specification.md#response-1
type GetHeaderResponse struct {
	Message   GetHeaderResponseMessage `json:"message"`
	Signature hexutil.Bytes            `json:"signature"`
}
