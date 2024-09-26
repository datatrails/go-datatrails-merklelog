package mmr

import "math/bits"

// MMRIndex returns the node index for the leaf e
//
// Args:
//   - leafIndex: the leaf index, where the leaves are numbered consequtively, ignoring interior nodes.
//
// Returns:
//
//	The mmr index for the element leafIndex
func MMRIndex(leafIndex uint64) uint64 {

	sum := uint64(0)
	for leafIndex > 0 {
		h := bits.Len64(leafIndex)
		sum += (1 << h) - 1
		half := 1 << (h - 1)
		leafIndex -= uint64(half)
	}
	return sum
}
