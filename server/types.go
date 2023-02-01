package server

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/attestantio/go-builder-client/api/capella"
	"github.com/attestantio/go-builder-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/flashbots/go-boost-utils/types"
)

var errNoResponse = errors.New("no response")

// wrapper for backwards compatible capella types

type GetHeaderResponse struct {
	Bellatrix *types.GetHeaderResponse
	Capella   *spec.VersionedSignedBuilderBid
}

func (r *GetHeaderResponse) UnmarshalJSON(data []byte) error {
	var err error

	var capella spec.VersionedSignedBuilderBid
	err = json.Unmarshal(data, &capella)
	if err == nil && capella.Capella != nil {
		r.Capella = &capella
		return nil
	}

	var bellatrix types.GetHeaderResponse
	err = json.Unmarshal(data, &bellatrix)
	if err != nil {
		return err
	}

	r.Bellatrix = &bellatrix

	return nil
}

func (r *GetHeaderResponse) MarshalJSON() ([]byte, error) {
	if r.Capella != nil {
		return json.Marshal(r.Capella)
	}

	if r.Bellatrix != nil {
		return json.Marshal(r.Bellatrix)
	}

	return nil, errNoResponse
}

func (r *GetHeaderResponse) IsInvalid() bool {
	if r.Bellatrix != nil {
		return r.Bellatrix.Data == nil || r.Bellatrix.Data.Message == nil || r.Bellatrix.Data.Message.Header == nil || r.Bellatrix.Data.Message.Header.BlockHash == nilHash
	}

	if r.Capella != nil {
		return r.Capella.Capella == nil || r.Capella.Capella.Message == nil || r.Capella.Capella.Message.Header == nil || r.Capella.Capella.Message.Header.BlockHash == phase0.Hash32(nilHash)
	}

	return true
}

func (r *GetHeaderResponse) BlockHash() string {
	if r.Bellatrix != nil {
		return r.Bellatrix.Data.Message.Header.BlockHash.String()
	}

	if r.Capella != nil {
		return r.Capella.Capella.Message.Header.BlockHash.String()
	}

	return ""
}

func (r *GetHeaderResponse) Value() *big.Int {
	if r.Bellatrix != nil {
		return r.Bellatrix.Data.Message.Value.BigInt()
	}

	if r.Capella != nil {
		return r.Capella.Capella.Message.Value.ToBig()
	}

	return nil
}

func (r *GetHeaderResponse) BlockNumber() uint64 {
	if r.Bellatrix != nil {
		return r.Bellatrix.Data.Message.Header.BlockNumber
	}

	if r.Capella != nil {
		return r.Capella.Capella.Message.Header.BlockNumber
	}

	return 0
}

func (r *GetHeaderResponse) TransactionsRoot() string {
	if r.Bellatrix != nil {
		return r.Bellatrix.Data.Message.Header.TransactionsRoot.String()
	}

	if r.Capella != nil {
		return r.Capella.Capella.Message.Header.TransactionsRoot.String()
	}

	return ""
}

func (r *GetHeaderResponse) Pubkey() string {
	if r.Bellatrix != nil {
		return r.Bellatrix.Data.Message.Pubkey.String()
	}

	if r.Capella != nil {
		return r.Capella.Capella.Message.Pubkey.String()
	}

	return ""
}

func (r *GetHeaderResponse) Signature() []byte {
	if r.Bellatrix != nil {
		return r.Bellatrix.Data.Signature[:]
	}

	if r.Capella != nil {
		return r.Capella.Capella.Signature[:]
	}

	return nil
}

func (r *GetHeaderResponse) Message() types.HashTreeRoot {
	if r.Bellatrix != nil {
		return r.Bellatrix.Data.Message
	}

	if r.Capella != nil {
		return r.Capella.Capella.Message
	}

	return nil
}

func (r *GetHeaderResponse) ParentHash() string {
	if r.Bellatrix != nil {
		return r.Bellatrix.Data.Message.Header.ParentHash.String()
	}

	if r.Capella != nil {
		return r.Capella.Capella.Message.Header.ParentHash.String()
	}

	return ""
}

func (r *GetHeaderResponse) IsEmpty() bool {
	return r.Bellatrix == nil && r.Capella == nil
}

func (r *GetHeaderResponse) BuilderBid() *SignedBuilderBid {
	if r.Bellatrix != nil {
		return &SignedBuilderBid{Bellatrix: r.Bellatrix.Data}
	}
	if r.Capella != nil {
		return &SignedBuilderBid{Capella: r.Capella.Capella}
	}
	return nil
}

type SignedBuilderBid struct {
	Bellatrix *types.SignedBuilderBid
	Capella   *capella.SignedBuilderBid
}

func (r *SignedBuilderBid) UnmarshalJSON(data []byte) error {
	var err error
	var bellatrix types.SignedBuilderBid
	err = json.Unmarshal(data, &bellatrix)
	if err == nil {
		r.Bellatrix = &bellatrix
		return nil
	}

	var capella capella.SignedBuilderBid
	err = json.Unmarshal(data, &capella)
	if err != nil {
		return err
	}

	r.Capella = &capella
	return nil
}

func (r *SignedBuilderBid) MarshalJSON() ([]byte, error) {
	if r.Bellatrix != nil {
		return json.Marshal(r.Bellatrix)
	}

	if r.Capella != nil {
		return json.Marshal(r.Capella)
	}

	return nil, errNoResponse
}
