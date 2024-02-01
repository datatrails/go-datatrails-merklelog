package mmr

import (
	"math"
	"math/bits"
)

// LeafCount returns the number of leaves in the largest mmr whose size is <=
// the supplied size. See also [triecommon/mmr/PeakBitmap]
func LeafCount(size uint64) uint64 {
	return PeaksBitmap(size)
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
