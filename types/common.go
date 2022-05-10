package types

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

type Signature [96]byte
type PublicKey [48]byte
type Address [20]byte
type Hash [32]byte
type Root Hash
type CommitteeBits [64]byte
type Bloom [256]byte
type U256Str Hash // encodes/decodes to string, not hex

var (
	ErrLength = fmt.Errorf("incorrect byte length")
)

func (s Signature) MarshalText() ([]byte, error) {
	return hexutil.Bytes(s[:]).MarshalText()
}

func (s *Signature) UnmarshalJSON(input []byte) error {
	b := hexutil.Bytes(s[:])
	err := b.UnmarshalJSON(input)
	if err != nil {
		return err
	}
	if len(b) != 96 {
		return ErrLength
	}
	s.FromSlice(b)
	return nil
}

func (s *Signature) UnmarshalText(input []byte) error {
	b := hexutil.Bytes(s[:])
	err := b.UnmarshalText(input)
	if err != nil {
		return err
	}
	if len(b) != 96 {
		return ErrLength
	}
	s.FromSlice(b)
	return nil

}

func (s Signature) String() string {
	return hexutil.Bytes(s[:]).String()
}

func (s *Signature) FromSlice(x []byte) {
	copy(s[:], x)
}

func (p PublicKey) MarshalText() ([]byte, error) {
	return hexutil.Bytes(p[:]).MarshalText()
}

func (p *PublicKey) UnmarshalJSON(input []byte) error {
	b := hexutil.Bytes(p[:])
	b.UnmarshalJSON(input)
	if len(b) != 48 {
		return ErrLength
	}
	p.FromSlice(b)
	return nil
}

func (p *PublicKey) UnmarshalText(input []byte) error {
	b := hexutil.Bytes(p[:])
	b.UnmarshalText(input)
	if len(b) != 48 {
		return ErrLength
	}
	p.FromSlice(b)
	return nil

}

func (p PublicKey) String() string {
	return hexutil.Bytes(p[:]).String()
}

func (p *PublicKey) FromSlice(x []byte) {
	copy(p[:], x)
}

func (a Address) MarshalText() ([]byte, error) {
	return hexutil.Bytes(a[:]).MarshalText()
}

func (a *Address) UnmarshalJSON(input []byte) error {
	b := hexutil.Bytes(a[:])
	b.UnmarshalJSON(input)
	if len(b) != 20 {
		return ErrLength
	}
	a.FromSlice(b)
	return nil
}

func (a *Address) UnmarshalText(input []byte) error {
	b := hexutil.Bytes(a[:])
	b.UnmarshalText(input)
	if len(b) != 20 {
		return ErrLength
	}
	a.FromSlice(b)
	return nil

}

func (a Address) String() string {
	return hexutil.Bytes(a[:]).String()
}

func (a *Address) FromSlice(x []byte) {
	copy(a[:], x)
}

func (h Hash) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

func (h *Hash) UnmarshalJSON(input []byte) error {
	b := hexutil.Bytes(h[:])
	b.UnmarshalJSON(input)
	if len(b) != 32 {
		return ErrLength
	}
	h.FromSlice(b)
	return nil
}

func (h *Hash) UnmarshalText(input []byte) error {
	b := hexutil.Bytes(h[:])
	b.UnmarshalText(input)
	if len(b) != 32 {
		return ErrLength
	}
	h.FromSlice(b)
	return nil

}

func (h *Hash) FromSlice(x []byte) {
	copy(h[:], x)
}

func (h Hash) String() string {
	return hexutil.Bytes(h[:]).String()
}

func (r Root) MarshalText() ([]byte, error) {
	return hexutil.Bytes(r[:]).MarshalText()
}

func (r *Root) UnmarshalJSON(input []byte) error {
	b := hexutil.Bytes(r[:])
	b.UnmarshalJSON(input)
	if len(b) != 32 {
		return ErrLength
	}
	r.FromSlice(b)
	return nil
}

func (r *Root) UnmarshalText(input []byte) error {
	b := hexutil.Bytes(r[:])
	b.UnmarshalText(input)
	if len(b) != 32 {
		return ErrLength
	}
	r.FromSlice(b)
	return nil
}

func (r *Root) FromSlice(x []byte) {
	copy(r[:], x)
}

func (r Root) String() string {
	return hexutil.Bytes(r[:]).String()
}

func (c CommitteeBits) MarshalText() ([]byte, error) {
	return hexutil.Bytes(c[:]).MarshalText()
}

func (c *CommitteeBits) UnmarshalJSON(input []byte) error {
	b := hexutil.Bytes(c[:])
	b.UnmarshalJSON(input)
	if len(b) != 64 {
		return ErrLength
	}
	c.FromSlice(b)
	return nil
}

func (c *CommitteeBits) UnmarshalText(input []byte) error {
	b := hexutil.Bytes(c[:])
	b.UnmarshalText(input)
	if len(b) != 64 {
		return ErrLength
	}
	c.FromSlice(b)
	return nil

}

func (c CommitteeBits) String() string {
	return hexutil.Bytes(c[:]).String()
}

func (c *CommitteeBits) FromSlice(x []byte) {
	copy(c[:], x)
}

func (b Bloom) MarshalText() ([]byte, error) {
	return hexutil.Bytes(b[:]).MarshalText()
}

func (b *Bloom) UnmarshalJSON(input []byte) error {
	buf := hexutil.Bytes(b[:])
	buf.UnmarshalJSON(input)
	if len(b) != 256 {
		return ErrLength
	}
	b.FromSlice(buf)
	return nil
}

func (b *Bloom) UnmarshalText(input []byte) error {
	buf := hexutil.Bytes(b[:])
	buf.UnmarshalText(input)
	if len(b) != 256 {
		return ErrLength
	}
	b.FromSlice(buf)
	return nil
}

func (b Bloom) String() string {
	return hexutil.Bytes(b[:]).String()
}

func (b *Bloom) FromSlice(x []byte) {
	copy(b[:], x)
}

func (n U256Str) MarshalText() ([]byte, error) {
	return []byte(new(big.Int).SetBytes(n[:]).String()), nil
}

func (n *U256Str) UnmarshalJSON(input []byte) error {
	if len(input) < 2 {
		return ErrLength
	}
	x := new(big.Int)
	err := x.UnmarshalJSON(input[1 : len(input)-1])
	if err != nil {
		return err
	}
	copy(n[:], x.FillBytes(n[:]))
	return nil
}

func (n *U256Str) UnmarshalText(input []byte) error {
	x := new(big.Int)
	err := x.UnmarshalText(input)
	if err != nil {
		return err
	}
	copy(n[:], x.FillBytes(n[:]))
	return nil

}

func (n *U256Str) String() string {
	return new(big.Int).SetBytes(n[:]).String()
}

func (n *U256Str) FromSlice(x []byte) {
	copy(n[:], x)
}

func IntToU256(i uint64) (ret U256Str) {
	s := fmt.Sprint(i)
	ret.UnmarshalText([]byte(s))
	return
}

func HexToAddress(s string) (ret Address) {
	ret.UnmarshalText([]byte(s))
	return
}

func HexToPubkey(s string) (ret PublicKey) {
	ret.UnmarshalText([]byte(s))
	return
}

func HexToSignature(s string) (ret Signature) {
	ret.UnmarshalText([]byte(s))
	return
}
