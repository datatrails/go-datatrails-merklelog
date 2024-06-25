package mmr

// LeafCount returns the number of leaves in the largest mmr whose size is <=
// the supplied size. See also [merklelog/mmr/PeakBitmap]
//
// This can safely be use to obtain the leaf index *only* when size is known to
// be a valid mmr size. Typically just before or just after calling AddHashedLeaf
// If in any doubt, instead do:
//
//	leafIndex = LeafCount(FirstMMRSize(mmrIndex)) -1
func LeafCount(size uint64) uint64 {
	return PeaksBitmap(size)
}

func LeafIndex(mmrIndex uint64) uint64 {
	return LeafCount(FirstMMRSize(mmrIndex)) - 1
}

// FirstMMRSize returns the first complete MMRSize that contains the provided
// mmrIndex. mmrIndices are used to identify nodes. mmrSizes are the result of
// *adding* nodes to mmr's, and, because of adding the back fill nodes for the
// leaves, the  range of valid sizes is not continuous. Typically, it is
// possible to "do the right thing" with just LeafCount, but its use is error
// prone because of this fact.
//
// The outputs of this function for the following mmrIndices are
//
//	[1, 3, 3, 4, 7, 7, 7, 8, 10, 10, 11]
//
//	2        6
//	       /   \
//	1     2     5      9
//	     / \   / \    / \
//	0   0   1 3   4  7   8 10
func FirstMMRSize(mmrIndex uint64) uint64 {

	i := mmrIndex
	h0 := IndexHeight(i)
	h1 := IndexHeight(i + 1)
	for h0 < h1 {
		i++
		h0 = h1
		h1 = IndexHeight(i + 1)
	}

	return i + 1
}
