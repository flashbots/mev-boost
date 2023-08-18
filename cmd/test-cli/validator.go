package main

import (
	"encoding/json"
	"errors"
	"os"
	"time"

	builderApiV1 "github.com/attestantio/go-builder-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/go-boost-utils/bls"
	"github.com/flashbots/go-boost-utils/ssz"
	"github.com/flashbots/go-boost-utils/utils"
)

var errInvalidLength = errors.New("invalid length")

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
	return validatorPrivateData{bls.SecretKeyToBytes(sk), bls.PublicKeyToBytes(pk), hexutil.Uint64(gasLimit), feeRecipient}
}

func (v *validatorPrivateData) PrepareRegistrationMessage(builderSigningDomain phase0.Domain) ([]builderApiV1.SignedValidatorRegistration, error) {
	pk := phase0.BLSPubKey{}
	if len(v.Pk) != len(pk) {
		return []builderApiV1.SignedValidatorRegistration{}, errInvalidLength
	}
	copy(pk[:], v.Pk)

	addr, err := utils.HexToAddress(v.FeeRecipientHex)
	if err != nil {
		return []builderApiV1.SignedValidatorRegistration{}, err
	}
	msg := builderApiV1.ValidatorRegistration{
		FeeRecipient: addr,
		Timestamp:    time.Now(),
		Pubkey:       pk,
		GasLimit:     uint64(v.GasLimit),
	}
	signature, err := v.Sign(&msg, builderSigningDomain)
	if err != nil {
		return []builderApiV1.SignedValidatorRegistration{}, err
	}

	return []builderApiV1.SignedValidatorRegistration{{
		Message:   &msg,
		Signature: signature,
	}}, nil
}

func (v *validatorPrivateData) Sign(msg ssz.ObjWithHashTreeRoot, domain phase0.Domain) (phase0.BLSSignature, error) {
	sk, err := bls.SecretKeyFromBytes(v.Sk)
	if err != nil {
		return phase0.BLSSignature{}, err
	}
	return ssz.SignMessage(msg, domain, sk)
}
