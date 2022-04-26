package types

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var NilHash = common.Hash{}

// SignedBlindedBeaconBlock forked from https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/phase0/beacon-chain.md#signedbeaconblockheader
type SignedBlindedBeaconBlock struct {
	Message   *BlindedBeaconBlock `json:"message"`
	Signature string              `json:"signature"`
}

// BlindedBeaconBlock forked from https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/phase0/beacon-chain.md#beaconblock
type BlindedBeaconBlock struct {
	Slot          string          `json:"slot"`
	ProposerIndex string          `json:"proposer_index"`
	ParentRoot    string          `json:"parent_root"`
	StateRoot     string          `json:"state_root"`
	Body          json.RawMessage `json:"body"`
}

// BlindedBeaconBlockBodyPartial a partial block body only containing a payload, in both snake_case and camelCase
type BlindedBeaconBlockBodyPartial struct {
	ExecutionPayload      ExecutionPayloadHeaderOnlyBlockHash `json:"execution_payload_header"`
	ExecutionPayloadCamel ExecutionPayloadHeaderOnlyBlockHash `json:"executionPayloadHeader"`
}

//go:generate go run github.com/fjl/gencodec -type ExecutionPayloadHeaderV1 -field-override executionPayloadMarshallingOverrides -out gen_executionpayloadheader.go
// ExecutionPayloadHeaderV1 as defined in https://github.com/flashbots/mev-boost/blob/main/docs/specification.md#executionpayloadheaderv1
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
// ExecutionPayloadV1 as defined in https://github.com/flashbots/mev-boost/blob/main/docs/specification.md#executionpayloadv1
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

// ExecutionPayloadHeaderOnlyBlockHash an execution payload with only a block hash, used for BlindedBeaconBlockBodyPartial
type ExecutionPayloadHeaderOnlyBlockHash struct {
	BlockHash      string `json:"block_hash"`
	BlockHashCamel string `json:"blockHash"`
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

// SetFeeRecipientMessage as defined in https://github.com/flashbots/mev-boost/blob/main/docs/specification.md#request
type SetFeeRecipientMessage struct {
	FeeRecipient string `json:"feeRecipient"`
	Timestamp    string `json:"timestamp"`
}

//go:generate go run github.com/fjl/gencodec -type GetHeaderResponseMessage -field-override getHeaderResponseMessageMarshallingOverrides -out gen_getheaderresponsemsg.go
// GetHeaderResponseMessage as defined in https://github.com/flashbots/mev-boost/blob/main/docs/specification.md#response-1
type GetHeaderResponseMessage struct {
	Header ExecutionPayloadHeaderV1 `json:"header" gencodec:"required"`
	Value  *big.Int                 `json:"value" gencodec:"required"`
}

//go:generate go run github.com/fjl/gencodec -type GetHeaderResponse -field-override getHeaderResponseMarshallingOverrides -out gen_getheaderresponse.go
// GetHeaderResponse as defined in https://github.com/flashbots/mev-boost/blob/main/docs/specification.md#response-1
type GetHeaderResponse struct {
	Message   GetHeaderResponseMessage `json:"message" gencodec:"required"`
	PublicKey []byte                   `json:"publicKey" gencodec:"required"`
	Signature []byte                   `json:"signature" gencodec:"required"`
}

type getHeaderResponseMessageMarshallingOverrides struct {
	Value *hexutil.Big
}

type getHeaderResponseMarshallingOverrides struct {
	PublicKey hexutil.Bytes
	Signature hexutil.Bytes
}
