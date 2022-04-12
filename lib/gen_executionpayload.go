// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package lib

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var _ = (*executionPayloadMarshallingOverrides)(nil)

// MarshalJSON marshals as JSON.
func (e ExecutionPayloadV1) MarshalJSON() ([]byte, error) {
	type ExecutionPayloadV1 struct {
		ParentHash    common.Hash    `json:"parentHash" gencodec:"required"`
		FeeRecipient  common.Address `json:"feeRecipient" gencodec:"required"`
		StateRoot     common.Hash    `json:"stateRoot" gencodec:"required"`
		ReceiptsRoot  common.Hash    `json:"receiptsRoot" gencodec:"required"`
		LogsBloom     hexutil.Bytes  `json:"logsBloom" gencodec:"required"`
		PrevRandao    common.Hash    `json:"prevRandao" gencodec:"required"`
		BlockNumber   hexutil.Uint64 `json:"blockNumber" gencodec:"required"`
		GasLimit      hexutil.Uint64 `json:"gasLimit" gencodec:"required"`
		GasUsed       hexutil.Uint64 `json:"gasUsed" gencodec:"required"`
		Timestamp     hexutil.Uint64 `json:"timestamp" gencodec:"required"`
		ExtraData     hexutil.Bytes  `json:"extraData" gencodec:"required"`
		BaseFeePerGas *hexutil.Big   `json:"baseFeePerGas" gencodec:"required"`
		BlockHash     common.Hash    `json:"blockHash" gencodec:"required"`
		Transactions  *[]string      `json:"transactions,omitempty"`
	}
	var enc ExecutionPayloadV1
	enc.ParentHash = e.ParentHash
	enc.FeeRecipient = e.FeeRecipient
	enc.StateRoot = e.StateRoot
	enc.ReceiptsRoot = e.ReceiptsRoot
	enc.LogsBloom = e.LogsBloom
	enc.PrevRandao = e.PrevRandao
	enc.BlockNumber = hexutil.Uint64(e.BlockNumber)
	enc.GasLimit = hexutil.Uint64(e.GasLimit)
	enc.GasUsed = hexutil.Uint64(e.GasUsed)
	enc.Timestamp = hexutil.Uint64(e.Timestamp)
	enc.ExtraData = e.ExtraData
	enc.BaseFeePerGas = (*hexutil.Big)(e.BaseFeePerGas)
	enc.BlockHash = e.BlockHash
	enc.Transactions = e.Transactions
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (e *ExecutionPayloadV1) UnmarshalJSON(input []byte) error {
	type ExecutionPayloadV1 struct {
		ParentHash    *common.Hash    `json:"parentHash" gencodec:"required"`
		FeeRecipient  *common.Address `json:"feeRecipient" gencodec:"required"`
		StateRoot     *common.Hash    `json:"stateRoot" gencodec:"required"`
		ReceiptsRoot  *common.Hash    `json:"receiptsRoot" gencodec:"required"`
		LogsBloom     *hexutil.Bytes  `json:"logsBloom" gencodec:"required"`
		PrevRandao    *common.Hash    `json:"prevRandao" gencodec:"required"`
		BlockNumber   *hexutil.Uint64 `json:"blockNumber" gencodec:"required"`
		GasLimit      *hexutil.Uint64 `json:"gasLimit" gencodec:"required"`
		GasUsed       *hexutil.Uint64 `json:"gasUsed" gencodec:"required"`
		Timestamp     *hexutil.Uint64 `json:"timestamp" gencodec:"required"`
		ExtraData     *hexutil.Bytes  `json:"extraData" gencodec:"required"`
		BaseFeePerGas *hexutil.Big    `json:"baseFeePerGas" gencodec:"required"`
		BlockHash     *common.Hash    `json:"blockHash" gencodec:"required"`
		Transactions  *[]string       `json:"transactions,omitempty"`
	}
	var dec ExecutionPayloadV1
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.ParentHash == nil {
		return errors.New("missing required field 'parentHash' for ExecutionPayloadV1")
	}
	e.ParentHash = *dec.ParentHash
	if dec.FeeRecipient == nil {
		return errors.New("missing required field 'feeRecipient' for ExecutionPayloadV1")
	}
	e.FeeRecipient = *dec.FeeRecipient
	if dec.StateRoot == nil {
		return errors.New("missing required field 'stateRoot' for ExecutionPayloadV1")
	}
	e.StateRoot = *dec.StateRoot
	if dec.ReceiptsRoot == nil {
		return errors.New("missing required field 'receiptsRoot' for ExecutionPayloadV1")
	}
	e.ReceiptsRoot = *dec.ReceiptsRoot
	if dec.LogsBloom == nil {
		return errors.New("missing required field 'logsBloom' for ExecutionPayloadV1")
	}
	e.LogsBloom = *dec.LogsBloom
	if dec.PrevRandao == nil {
		return errors.New("missing required field 'prevRandao' for ExecutionPayloadV1")
	}
	e.PrevRandao = *dec.PrevRandao
	if dec.BlockNumber == nil {
		return errors.New("missing required field 'blockNumber' for ExecutionPayloadV1")
	}
	e.BlockNumber = uint64(*dec.BlockNumber)
	if dec.GasLimit == nil {
		return errors.New("missing required field 'gasLimit' for ExecutionPayloadV1")
	}
	e.GasLimit = uint64(*dec.GasLimit)
	if dec.GasUsed == nil {
		return errors.New("missing required field 'gasUsed' for ExecutionPayloadV1")
	}
	e.GasUsed = uint64(*dec.GasUsed)
	if dec.Timestamp == nil {
		return errors.New("missing required field 'timestamp' for ExecutionPayloadV1")
	}
	e.Timestamp = uint64(*dec.Timestamp)
	if dec.ExtraData == nil {
		return errors.New("missing required field 'extraData' for ExecutionPayloadV1")
	}
	e.ExtraData = *dec.ExtraData
	if dec.BaseFeePerGas == nil {
		return errors.New("missing required field 'baseFeePerGas' for ExecutionPayloadV1")
	}
	e.BaseFeePerGas = (*big.Int)(dec.BaseFeePerGas)
	if dec.BlockHash == nil {
		return errors.New("missing required field 'blockHash' for ExecutionPayloadV1")
	}
	e.BlockHash = *dec.BlockHash
	if dec.Transactions != nil {
		e.Transactions = dec.Transactions
	}
	return nil
}
