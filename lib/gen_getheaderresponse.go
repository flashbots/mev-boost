// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package lib

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

var _ = (*getHeaderResponseMarshallingOverrides)(nil)

// MarshalJSON marshals as JSON.
func (g GetHeaderResponse) MarshalJSON() ([]byte, error) {
	type GetHeaderResponse struct {
		Header    ExecutionPayloadHeaderV1 `json:"header" gencodec:"required"`
		Value     *hexutil.Big             `json:"value" gencodec:"required"`
		PublicKey hexutil.Bytes            `json:"publicKey" gencodec:"required"`
		Signature hexutil.Bytes            `json:"signature" gencodec:"required"`
	}
	var enc GetHeaderResponse
	enc.Header = g.Header
	enc.Value = (*hexutil.Big)(g.Value)
	enc.PublicKey = g.PublicKey
	enc.Signature = g.Signature
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (g *GetHeaderResponse) UnmarshalJSON(input []byte) error {
	type GetHeaderResponse struct {
		Header    *ExecutionPayloadHeaderV1 `json:"header" gencodec:"required"`
		Value     *hexutil.Big              `json:"value" gencodec:"required"`
		PublicKey *hexutil.Bytes            `json:"publicKey" gencodec:"required"`
		Signature *hexutil.Bytes            `json:"signature" gencodec:"required"`
	}
	var dec GetHeaderResponse
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Header == nil {
		return errors.New("missing required field 'header' for GetHeaderResponse")
	}
	g.Header = *dec.Header
	if dec.Value == nil {
		return errors.New("missing required field 'value' for GetHeaderResponse")
	}
	g.Value = (*big.Int)(dec.Value)
	if dec.PublicKey == nil {
		return errors.New("missing required field 'publicKey' for GetHeaderResponse")
	}
	g.PublicKey = *dec.PublicKey
	if dec.Signature == nil {
		return errors.New("missing required field 'signature' for GetHeaderResponse")
	}
	g.Signature = *dec.Signature
	return nil
}