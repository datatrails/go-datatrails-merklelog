package mmr

import (
	"encoding/hex"
	"fmt"
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
func proofPathsStringer(paths [][][]byte, sep string) string {

	spaths := make([]string, 0, len(paths))

	for _, path := range paths {
		spaths = append(spaths, fmt.Sprintf("[%s]", proofPathStringer(path, sep)))
	}
	return strings.Join(spaths, sep)
}
