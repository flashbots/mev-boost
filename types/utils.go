package types

import (
	"strings"
)

// IsValidHex takes a hex string and validates that it is hexadecimal, with 0x prefix, containing `size` number of bytes.
func IsValidHex(value string, size int) bool {
	if !strings.HasPrefix(value, "0x") {
		return false
	}

	if size != -1 {
		dataSize := (len(value) - 2) / 2

		if size != dataSize {
			return false
		}
	}

	// validate that the input characters after 0x are only 0-9, a-f, A-F
	for _, c := range value[2:] {
		switch {
		case '0' <= c && c <= '9':
			continue
		case 'a' <= c && c <= 'f':
			continue
		case 'A' <= c && c <= 'F':
			continue
		}

		return false
	}

	return true
}
