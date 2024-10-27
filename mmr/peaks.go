package mmr

import (
	"math/bits"
)

// Peaks returns the array of mountain peak indices in the MMR.
//
// This is completely deterministic given a complete mmr index.
// If the mmr index is not complete, or is otherwise invalid, is invalid, this function returns nil.
//
// The peaks are listed in ascending order of mmr index value.
// The highest peak has the lowest index and is listed first. This is a
// consequence of the fact that the 'little' 'down range' peaks can only appear
// to the 'right' of the first perfect peak, and so on recursively.
//
// Given the example below, which has an mmrSize of 10, the peaks are [6, 9]:
//
//	2        6
//	       /   \
//	1     2     5      9
//	     / \   / \    / \
//	0   0   1 3   4  7   8
func Peaks(mmrIndex uint64) []uint64 {

	// The peaks algorithm works using the binary properties of the mmr *positions*

	mmrSize := mmrIndex + 1

	// catch invalid range, where siblings exist but no parent exists
	if PosHeight(mmrSize+1) > PosHeight(mmrSize) {
		return nil
	}

	peak := uint64(0)
	var peaks []uint64
	// The top peak is always the left most and, when counting from 1, will have all binary '1's
	for mmrSize != 0 {
		// This next step computes the ^2 floor of the bits in mmrSize, which
		// picks out the highest peak (and also left most) remaining peak in
		// mmrSize (See TopPeak)
		peakSize := TopPeak(mmrSize-1) + 1 // + 1 to recover position form

		// Because we *subtract* the computed peak size from mmrSize, we need to
		// recover the actual peak position. The arithmetic all works out so we
		// just accumulate the peakSizes as we go, and the result is always the
		// peak value against the original mmrSize we were given.
		peak = peak + peakSize
		peaks = append(peaks, peak-1)
		mmrSize -= peakSize
	}
	return peaks
}

// PosPeaks is a depricated version of peaks which returns an array of mmr positions rather than indices.
func PosPeaks(mmrSize uint64) []uint64 {

	peaks := Peaks(mmrSize - 1)
	if peaks == nil {
		return nil
	}
	for i, p := range peaks {
		peaks[i] = p + 1
	}
	return peaks
}

func PeakHashes(store indexStoreGetter, mmrIndex uint64) ([][]byte, error) {
	// Note: we can implement this directly any time we want, but lets re-use the testing for Peaks
	var path [][]byte
	for _, i := range Peaks(mmrIndex) {
		stored, err := store.Get(i)
		if err != nil {
			return nil, err
		}

		value := make([]byte, 32)
		copy(value, stored)

		// Note: we create a copy here to ensure the value is not modified under the callers feet
		path = append(path, value)
	}
	return path, nil
}

// PeakIndex returns the index of the peak accumulator for the peak with the provided proof length.
//
// Given:
//
//	leafCount - the count of elements in the current accumulator, eg LeafCount(mmrIndex).
//	d - the length of the proof of any element in the mmr identified by leafCount
//
// Return
//
//	The index of the accumulator peak produced by a valid inclusion proof of length d
//
// Note that leafCount identifies the mmr state, not the element.
//
// For interior nodes, you must account for the height by adding IndexHeigh(mmrIndex) to the proof length d.
//
// Example:
//
//		peaks = PosPeaks(18) = [14, 17]
//		peakBits = LeafCount(18) = 101
//	 	1 = d = proof len for 6
//		2 = IndexHeight(6)
//		peaks[PeakIndex(peakBits, 1 + 2)] == 14
//
// For this MMR:
//
//	3              14
//	             /    \
//	            /      \
//	           /        \
//	          /          \
//	2        6            13
//	       /   \        /    \
//	1     2     5      9     12     17
//	     / \   / \    / \   /  \   /  \
//	0   0   1 3   4  7   8 10  11 15  16
func PeakIndex(leafCount uint64, d int) int {

	// The bitmask corresponding to the peaks in the accumulator is the leaf
	// index e + 1, which is leafCount.
	// The inclusion proof depth for any element is always the index
	// of a set bit in this mask.
	// And the bit corresponds to the peak which is the root for the element who's inclusion is proven.

	peaksMask := uint64(1<<(d+1) - 1)

	// The count of non zero bits
	n := bits.OnesCount64(leafCount & peaksMask)

	// We are ajusting to account for the gaps removed from the accumulator in
	// our 'packed' representation.  but the algerbra just works out so we index
	// by the number of set bits.

	// A[d - (d - n) - 1] = A[d -d + n -1] = A[n-1]

	// Now account for the fact that the accumulator lists peaks highest to lowest
	// So we need to invert the index

	// The accumulator length a is just the number of bits set in the leaf count

	// (a - 1) - (n -1) = a - 1 - n + 1 = a - n

	return int(bits.OnesCount64(leafCount)) - n
}

// TopPeak returns the smallest, leftmost, peak containing *or equal to* i
//
// This is essentially a ^2 *floor* function for the accumulation of bits:
//
//	TopPeak(0) = TopPeak(1) = 0
//	TopPeak(1) = TopPeak(2) = TopPeak(3) = TopPeak(4) = TopPeak(5) = 2
//	TopPeak(6) = 6
//
//	2       6
//	      /   \
//	1    2     5     9
//	    / \  /  \   / \
//	0  0   1 3   4 7   8 10
func TopPeak(i uint64) uint64 {

	// This works by working out the next peak *position* up then subtracting 1, which is a
	// flooring function for the bits over the current peak
	return 1<<(BitLength64(i+2)-1) - 2
}

// TopHeight returns the index height of the largest perfect peak contained in, or exactly, pos
// This is essentially a ^2 *floor* function for the accumulation of bits:
//
//	TopHeight(0) = TopHeight(1) = 0
//	TopHeight(1) = TopHeight(2) = TopHeight(3) = TopHeight(4) = TopHeight(5) = 1
//	TopHeight(6) = 2
//
//	2       6
//	      /   \
//	1    2     5     9
//	    / \  /  \   / \
//	0  0   1 3   4 7   8 10
func TopHeight(i uint64) uint64 {
	return BitLength64(i+2) - 2
}

// PeaksBitmap returns a bit mask where a 1 corresponds to a peak and the position
// of the bit is the height of that peak. The resulting value is also the count
// of leaves. This is due to the binary nature of the tree.
//
// For example, with an mmr with size 19, there are 11 leaves
//
//	          14
//	       /       \
//	     6          13
//	   /   \       /   \
//	  2     5     9     12     17
//	 / \   /  \  / \   /  \   /  \
//	0   1 3   4 7   8 10  11 15  16 18
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
	// peakSize := uint64(math.MaxUint64) >> bits.LeadingZeros64(mmrSize)
	peakSize := (uint64(1) << bits.Len64(mmrSize)) - 1
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
