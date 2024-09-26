package mmr

// LeafCount returns the number of leaves in the largest mmr whose size is <=
// the supplied size. See also [merklelog/mmr/PeakBitmap]
//
// This can safely be use to obtain the leaf index *only* when size is known to
// be a valid mmr size. Typically just before or just after calling AddHashedLeaf
// If in any doubt, instead use LeafIndex() + 1
func LeafCount(size uint64) uint64 {
	return PeaksBitmap(size)
}

func LeafIndex(mmrIndex uint64) uint64 {
	return LeafCount(FirstMMRSize(mmrIndex)) - 1
}
