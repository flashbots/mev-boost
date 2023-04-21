package server

import (
	"encoding/json"
	"errors"
	"math/big"

	builderApiBellatrix "github.com/attestantio/go-builder-client/api/bellatrix"
	builderApiCapella "github.com/attestantio/go-builder-client/api/capella"
	builderSpec "github.com/attestantio/go-builder-client/spec"
	"github.com/flashbots/go-boost-utils/ssz"
)

var errNoResponse = errors.New("no response")

// wrapper for backwards compatible capella types

type GetHeaderResponse builderSpec.VersionedSignedBuilderBid

func (r *GetHeaderResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*builderSpec.VersionedSignedBuilderBid)(r))
}

func (r *GetHeaderResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(&builderSpec.VersionedSignedBuilderBid{
		Version:   r.Version,
		Bellatrix: r.Bellatrix,
		Capella:   r.Capella,
	})
}

func (r *GetHeaderResponse) IsInvalid() bool {
	if r.Bellatrix != nil {
		return r.Bellatrix == nil || r.Bellatrix.Message == nil || r.Bellatrix.Message.Header == nil || r.Bellatrix.Message.Header.BlockHash == nilHash
	}

	if r.Capella != nil {
		return r.Capella == nil || r.Capella.Message == nil || r.Capella.Message.Header == nil || r.Capella.Message.Header.BlockHash == nilHash
	}

	return true
}

func (r *GetHeaderResponse) BlockHash() string {
	if r.Bellatrix != nil {
		return r.Bellatrix.Message.Header.BlockHash.String()
	}

	if r.Capella != nil {
		return r.Capella.Message.Header.BlockHash.String()
	}

	return ""
}

func (r *GetHeaderResponse) Value() *big.Int {
	if r.Bellatrix != nil {
		return r.Bellatrix.Message.Value.ToBig()
	}

	if r.Capella != nil {
		return r.Capella.Message.Value.ToBig()
	}

	return nil
}

func (r *GetHeaderResponse) BlockNumber() uint64 {
	if r.Bellatrix != nil {
		return r.Bellatrix.Message.Header.BlockNumber
	}

	if r.Capella != nil {
		return r.Capella.Message.Header.BlockNumber
	}

	return 0
}

func (r *GetHeaderResponse) TransactionsRoot() string {
	if r.Bellatrix != nil {
		return r.Bellatrix.Message.Header.TransactionsRoot.String()
	}

	if r.Capella != nil {
		return r.Capella.Message.Header.TransactionsRoot.String()
	}

	return ""
}

func (r *GetHeaderResponse) Pubkey() string {
	if r.Bellatrix != nil {
		return r.Bellatrix.Message.Pubkey.String()
	}

	if r.Capella != nil {
		return r.Capella.Message.Pubkey.String()
	}

	return ""
}

func (r *GetHeaderResponse) Signature() []byte {
	if r.Bellatrix != nil {
		return r.Bellatrix.Signature[:]
	}

	if r.Capella != nil {
		return r.Capella.Signature[:]
	}

	return nil
}

func (r *GetHeaderResponse) Message() ssz.ObjWithHashTreeRoot {
	if r.Bellatrix != nil {
		return r.Bellatrix.Message
	}

	if r.Capella != nil {
		return r.Capella.Message
	}

	return nil
}

func (r *GetHeaderResponse) ParentHash() string {
	if r.Bellatrix != nil {
		return r.Bellatrix.Message.Header.ParentHash.String()
	}

	if r.Capella != nil {
		return r.Capella.Message.Header.ParentHash.String()
	}

	return ""
}

func (r *GetHeaderResponse) IsEmpty() bool {
	return r.Bellatrix == nil && r.Capella == nil
}

func (r *GetHeaderResponse) BuilderBid() *SignedBuilderBid {
	if r.Bellatrix != nil {
		return &SignedBuilderBid{Bellatrix: r.Bellatrix}
	}
	if r.Capella != nil {
		return &SignedBuilderBid{Capella: r.Capella}
	}
	return nil
}

type SignedBuilderBid struct {
	Bellatrix *builderApiBellatrix.SignedBuilderBid
	Capella   *builderApiCapella.SignedBuilderBid
}

func (r *SignedBuilderBid) UnmarshalJSON(data []byte) error {
	var capella builderApiCapella.SignedBuilderBid
	err := json.Unmarshal(data, &capella)
	if err != nil {
		r.Capella = &capella
		return err
	}

	var bellatrix builderApiBellatrix.SignedBuilderBid
	err = json.Unmarshal(data, &bellatrix)
	if err == nil {
		r.Bellatrix = &bellatrix
		return nil
	}

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
