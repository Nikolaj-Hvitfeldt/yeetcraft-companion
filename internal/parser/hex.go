package parser

import (
	"strconv"
	"strings"
)

// parseHexUint32 parses the explicit hexadecimal token form used by V22.
func parseHexUint32(token string) (uint32, bool) {
	if len(token) <= 2 || (!strings.HasPrefix(token, "0x") && !strings.HasPrefix(token, "0X")) {
		return 0, false
	}
	value, err := strconv.ParseUint(token[2:], 16, 32)
	if err != nil {
		return 0, false
	}
	return uint32(value), true
}
