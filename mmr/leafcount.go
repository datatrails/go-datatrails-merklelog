package mmr

import (
	"math"
	"math/bits"
)

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

func LeafIndex(mmrIndex uint64) uint64 {
	return LeafCount(FirstMMRSize(mmrIndex)) - 1
}

// PeakMap returns a bit mask where a 1 corresponds to a peak and the position
// of the bit is the height of that peak. The resulting value is also the count
// of leaves. This is due to the binary nature of the tree.
//
// For example, with an mmr with size 19, there are 11 leaves
//
//	         14
//	      /       \
//	    6          13
//	  /   \       /   \
//	 2     5     9     12     17
//	/ \   /  \  / \   /  \   /  \
//
// 0   1 3   4 7   8 10  11 15  16 18
//
// PeakMap(19) returns 0b1011 which shows, reading from the right (low bit),
// there are peaks, that the lowest peak is at height 0, the second lowest at
// height 1, then the next and last peak is at height 3.
//
// If the provided mmr size is invalid, the returned map will be for the largest
// valid mmr size < the provided invalid size.
func PeaksBitmap(mmrSize uint64) uint64 {
	if mmrSize == 0 {
		return 0
	}
	pos := mmrSize
	peakSize := uint64(math.MaxUint64) >> bits.LeadingZeros64(mmrSize)
	peakMap := uint64(0)
	for peakSize > 0 {
		peakMap <<= 1
		if pos >= peakSize {
			pos -= peakSize
			peakMap |= 1
		}
		peakSize >>= 1
	}
	return peakMap
}
