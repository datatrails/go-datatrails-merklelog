package massifs

// TreeRootIndex returns the root index for the tree with height
func TreeRootIndex(height uint8) uint64 {
	return (1 << height) - 2
}

// RangeRootIndex return the Massif root node's mmr index in the overall MMR  given
// the massif height and the first index of the MMR it contains
func RangeRootIndex(firstIndex uint64, height uint8) uint64 {
	return firstIndex + (1 << height) - 2
}

// RangeLastLeafIndex returns the mmr index of the last leaf given the first
// index of a massif and its height.
func RangeLastLeafIndex(firstIndex uint64, height uint8) uint64 {
	return firstIndex + TreeLastLeafIndex(height)
}

// TreeLastLeafIndex returns the *MMR* index of the last leaf in the tree with
// the given height (1 << h) - h -1 works because the number of nodes required
// to include the last leaf is always equal to the MMR height produced by node
func TreeLastLeafIndex(height uint8) uint64 {
	return (1 << height) - uint64(height) - 1
}

// TreeSize returns the maximum byte size of the tree based on the defined log
// entry size
func TreeSize(height uint8) uint64 {
	return TreeCount(height) * LogEntryBytes
}

//  TreeCount returns the node count
func TreeCount(height uint8) uint64 {
	return ((1 << height) - 1)
}
