package mmr

// HeightIndexSize returns the node count corresponding to the zero based height index
func HeightIndexSize(heightIndex uint64) uint64 {
	return (2 << heightIndex) - 1
}

// HeightMaxIndex returns the node index corresponding to the zero based height index
func HeightMaxIndex(heightIndex uint64) uint64 {
	return (2 << heightIndex) - 2
}

// HeightSize returns the size of the mmr with the provided height
func HeightSize(heightIndex uint64) uint64 {
	return (1 << heightIndex) - 1
}
