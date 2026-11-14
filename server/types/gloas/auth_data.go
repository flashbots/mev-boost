package gloas

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strings"

	builderApiGloas "github.com/attestantio/go-builder-client/api/gloas"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const MaxBuilderAuthDataSize = builderApiGloas.MaxBuilderAuthDataSize

var (
	ErrEmptyAuthData    = errors.New("auth data must not be empty")
	ErrAuthDataTooLarge = fmt.Errorf("auth data exceeds %d bytes", MaxBuilderAuthDataSize)
	ErrMissingHostname  = errors.New("url has no hostname")
	ErrNilMessage       = errors.New("nil message")
)

func ParseAuthData(s string) ([]byte, error) {
	data := []byte(s)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		var err error
		if data, err = hexutil.Decode(s); err != nil {
			return nil, err
		}
	}
	if len(data) == 0 {
		return nil, ErrEmptyAuthData
	}
	if len(data) > MaxBuilderAuthDataSize {
		return nil, ErrAuthDataTooLarge
	}
	return data, nil
}

func DefaultAuthData(rawURL string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return nil, ErrMissingHostname
	}
	if strings.Contains(host, ":") {
		addr, err := netip.ParseAddr(host)
		if err != nil {
			return nil, fmt.Errorf("invalid ipv6 literal %q: %w", host, err)
		}
		host = "[" + compressIPv6(addr) + "]"
	}
	return []byte(host), nil
}

func compressIPv6(addr netip.Addr) string {
	if !addr.Is4In6() {
		return addr.String()
	}
	b := addr.As16()
	// bytes 12-15 are the ipv4 address
	return fmt.Sprintf("::ffff:%x:%x",
		binary.BigEndian.Uint16(b[12:]), // reads bytes 12 and 13 as 1 group
		binary.BigEndian.Uint16(b[14:]), // reads bytes 14 and 15 as 1 group
	)
}
