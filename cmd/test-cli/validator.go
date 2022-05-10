package main

import (
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/go-boost-utils/bls"
	boostTypes "github.com/flashbots/go-boost-utils/types"
)

type validatorPrivateData struct {
	Sk          hexutil.Bytes
	Pk          hexutil.Bytes
	GasLimit    hexutil.Uint64
	CoinbaseHex string
}

func (v *validatorPrivateData) SaveValidator(filePath string) error {
	validatorData, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, validatorData, 0644)
}

func mustLoadValidator(filePath string) validatorPrivateData {
	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.WithField("filePath", filePath).WithField("err", err).Fatal("Could not load validator data")
	}
	var v validatorPrivateData
	err = json.Unmarshal(fileData, &v)
	if err != nil {
		log.WithField("filePath", filePath).WithField("fileData", fileData).WithField("err", err).Fatal("Could not parse validator data")
	}
	return v
}

func newRandomValidator(gasLimit uint64, coinbase string) validatorPrivateData {
	sk, pk, err := bls.GenerateNewKeypair()
	if err != nil {
		log.WithError(err).Fatal("unable to generate bls key pair")
	}
	return validatorPrivateData{sk.Serialize(), pk.Compress(), hexutil.Uint64(gasLimit), coinbase}
}

func (v *validatorPrivateData) PrepareRegistrationMessage() (boostTypes.SignedValidatorRegistration, error) {
	pk := boostTypes.PublicKey{}
	pk.FromSlice(v.Pk)
	addr, err := boostTypes.HexToAddress(v.CoinbaseHex)
	if err != nil {
		return boostTypes.SignedValidatorRegistration{}, err
	}
	msg := boostTypes.RegisterValidatorRequestMessage{
		FeeRecipient: addr,
		Timestamp:    uint64(time.Now().UnixMilli()),
		Pubkey:       pk,
		GasLimit:     uint64(v.GasLimit),
	}
	signature, err := v.Sign(&msg, boostTypes.DomainBuilder)
	log.WithField("msg", msg).WithField("domain", boostTypes.DomainBuilder).WithField("pk", pk).WithField("sig", signature).Info("register V")
	if err != nil {
		return boostTypes.SignedValidatorRegistration{}, err
	}

	return boostTypes.SignedValidatorRegistration{
		Message:   &msg,
		Signature: signature,
	}, nil
}

func (v *validatorPrivateData) Sign(msg boostTypes.HashTreeRoot, domain boostTypes.Domain) (boostTypes.Signature, error) {
	sk, err := bls.SecretKeyFromBytes(v.Sk)
	if err != nil {
		return boostTypes.Signature{}, err
	}
	return boostTypes.SignMessage(msg, domain, sk)
}
