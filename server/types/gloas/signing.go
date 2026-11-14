package gloas

import (
	builderApiGloas "github.com/attestantio/go-builder-client/api/gloas"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/flashbots/go-boost-utils/bls"
)

// DomainTypeBuilderRequestAuth is DOMAIN_BUILDER_REQUEST_AUTH.
var DomainTypeBuilderRequestAuth = phase0.DomainType{0x0b, 0x00, 0x00, 0x01}

// Verify checks the proposer's signature on a request auth.
func Verify(signed *builderApiGloas.SignedBuilderRequestAuth, pubkey phase0.BLSPubKey, domain phase0.Domain) (bool, error) {
	if signed == nil || signed.Message == nil {
		return false, ErrNilMessage
	}
	root, err := signingRoot(signed.Message, domain)
	if err != nil {
		return false, err
	}
	return bls.VerifySignatureBytes(root[:], signed.Signature[:], pubkey[:])
}

func signingRoot(auth *builderApiGloas.BuilderRequestAuth, domain phase0.Domain) (phase0.Root, error) {
	root, err := auth.HashTreeRoot()
	if err != nil {
		return phase0.Root{}, err
	}
	signingData := phase0.SigningData{ObjectRoot: root, Domain: domain}
	return signingData.HashTreeRoot()
}
