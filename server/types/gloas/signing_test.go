package gloas

import (
	"testing"

	builderApiGloas "github.com/attestantio/go-builder-client/api/gloas"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/flashbots/go-boost-utils/bls"
	"github.com/flashbots/go-boost-utils/ssz"
	"github.com/stretchr/testify/require"
)

func testDomain(forkVersion phase0.Version) phase0.Domain {
	return ssz.ComputeDomain(DomainTypeBuilderRequestAuth, forkVersion, phase0.Root{})
}

func testSign(t *testing.T, msg *builderApiGloas.BuilderRequestAuth, sk *bls.SecretKey, domain phase0.Domain) phase0.BLSSignature {
	t.Helper()
	root, err := ssz.ComputeSigningRoot(msg, domain)
	require.NoError(t, err)
	var sig phase0.BLSSignature
	copy(sig[:], bls.SignatureToBytes(bls.Sign(sk, root[:])))
	return sig
}

func TestSignAndVerify(t *testing.T) {
	sk, pk, err := bls.GenerateNewKeypair()
	require.NoError(t, err)
	var pubkey phase0.BLSPubKey
	copy(pubkey[:], bls.PublicKeyToBytes(pk))

	domain := testDomain(phase0.Version{})
	require.Equal(t, [4]byte{0x0b, 0x00, 0x00, 0x01}, [4]byte(domain[:4]))

	msg := &builderApiGloas.BuilderRequestAuth{Data: []byte("relay.example.com"), Slot: 42}
	sig := testSign(t, msg, sk, domain)
	signed := &builderApiGloas.SignedBuilderRequestAuth{Message: msg, Signature: sig}

	ok, err := Verify(signed, pubkey, domain)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = Verify(signed, pubkey, testDomain(phase0.Version{0x01}))
	require.NoError(t, err)
	require.False(t, ok, "wrong fork version")

	tampered := &builderApiGloas.SignedBuilderRequestAuth{Message: &builderApiGloas.BuilderRequestAuth{Data: msg.Data, Slot: 43}, Signature: sig}
	ok, err = Verify(tampered, pubkey, domain)
	require.NoError(t, err)
	require.False(t, ok, "tampered slot")

	_, otherPk, err := bls.GenerateNewKeypair()
	require.NoError(t, err)
	var otherPubkey phase0.BLSPubKey
	copy(otherPubkey[:], bls.PublicKeyToBytes(otherPk))
	ok, err = Verify(signed, otherPubkey, domain)
	require.NoError(t, err)
	require.False(t, ok, "other key")

	_, err = Verify(&builderApiGloas.SignedBuilderRequestAuth{}, pubkey, domain)
	require.ErrorIs(t, err, ErrNilMessage)
}
