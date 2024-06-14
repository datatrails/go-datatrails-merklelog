package mmr

import (
	"encoding/hex"
	"strings"
)

// debug utilities

func proofPathStringer(path [][]byte, sep string) string {
	var spath []string

	for _, it := range path {
		spath = append(spath, hex.EncodeToString(it))
	}
	return strings.Join(spath, sep)
}
