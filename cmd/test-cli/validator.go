package main

import (
	"encoding/json"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/go-boost-utils/bls"
	boostTypes "github.com/flashbots/go-boost-utils/types"
)

type validatorPrivateData struct {
	Sk              hexutil.Bytes
	Pk              hexutil.Bytes
	GasLimit        hexutil.Uint64
	FeeRecipientHex string
}

func (v *validatorPrivateData) SaveValidator(filePath string) error {
	validatorData, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, validatorData, 0o644)
}

func mustLoadValidator(filePath string) validatorPrivateData {
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		log.WithField("filePath", filePath).WithError(err).Fatal("Could not load validator data")
	}
	var v validatorPrivateData
	err = json.Unmarshal(fileData, &v)
	if err != nil {
		log.WithField("filePath", filePath).WithField("fileData", fileData).WithError(err).Fatal("Could not parse validator data")
	}
	return v
}

func newRandomValidator(gasLimit uint64, feeRecipient string) validatorPrivateData {
	sk, pk, err := bls.GenerateNewKeypair()
	if err != nil {
		log.WithError(err).Fatal("unable to generate bls key pair")
	}
	return validatorPrivateData{sk.Serialize(), pk.Compress(), hexutil.Uint64(gasLimit), feeRecipient}
}

func (v *validatorPrivateData) PrepareRegistrationMessage(builderSigningDomain boostTypes.Domain) ([]boostTypes.SignedValidatorRegistration, error) {
	pk := boostTypes.PublicKey{}
	err := pk.FromSlice(v.Pk)
	if err != nil {
		return []boostTypes.SignedValidatorRegistration{}, err
	}

	addr, err := boostTypes.HexToAddress(v.FeeRecipientHex)
	if err != nil {
		return []boostTypes.SignedValidatorRegistration{}, err
	}
	msg := boostTypes.RegisterValidatorRequestMessage{
		FeeRecipient: addr,
		Timestamp:    uint64(time.Now().Unix()),
		Pubkey:       pk,
		GasLimit:     uint64(v.GasLimit),
	}
	signature, err := v.Sign(&msg, builderSigningDomain)
	if err != nil {
		return []boostTypes.SignedValidatorRegistration{}, err
	}

	return []boostTypes.SignedValidatorRegistration{{
		Message:   &msg,
		Signature: signature,
	}}, nil
}

func (v *validatorPrivateData) Sign(msg boostTypes.HashTreeRoot, domain boostTypes.Domain) (boostTypes.Signature, error) {
	sk, err := bls.SecretKeyFromBytes(v.Sk)
	if err != nil {
		return boostTypes.Signature{}, err
	}
	return boostTypes.SignMessage(msg, domain, sk)
}
