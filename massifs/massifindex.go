package massifs

import "github.com/datatrails/go-datatrails-merklelog/mmr"

// MassifIndexFromLeafIndex gets the massif index of the massif that the given leaf is stored in,
//
//	given the leaf index of the leaf.
//
// This is found with the given massif height, which is constant for all massifs.
func MassifIndexFromLeafIndex(massifHeight uint8, leafIndex uint64) uint64 {

	// first find how many leaf nodes each massif can hold.
	//
	// Note: massifHeight starts at index 1, whereas height index for HeighIndexLeafCount starts at 0.
	massifMaxLeaves := mmr.HeightIndexLeafCount(uint64(massifHeight) - 1)

	// now find the massif.
	//
	// for context, see: https://github.com/datatrails/epic-8120-scalable-proof-mechanisms/blob/main/mmr/forestrie-mmrblobs.md#blob-size
	//
	// Note: massif indexes start at 0.
	// Note: leaf indexes starts at 0.
	//
	// Therefore, given a massif height of 2, that has max leaves of 4;
	//  if a leaf index of 3 is given, then it is in massif 0, along with leaves, 0, 1 and 2.
	return leafIndex / massifMaxLeaves

}

// MassifIndexFromMMRIndex gets the massif index of the massif that the given leaf is stored in
//
//	given the mmr index of the leaf.
//
// NOTE: if the mmrIndex is not a leaf node, then error is returned.
func MassifIndexFromMMRIndex(massifHeight uint8, mmrIndex uint64) uint64 {

	leafIndex := mmr.LeafIndex(mmrIndex)

	return MassifIndexFromLeafIndex(massifHeight, leafIndex)

}

// MassifFromLeaf computes the massif index given a leaf index and the configured massif height (one based) for the log.
func MassifFromLeaf(massifHeight uint8, leafIndex uint64) uint64 {

	// Note: massifHeight starts at index 1, whereas height index for HeighIndexLeafCount starts at 0.
	massifMaxLeaves := mmr.HeightIndexLeafCount(uint64(massifHeight) - 1)
	return leafIndex / massifMaxLeaves
}
