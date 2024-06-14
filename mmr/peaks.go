package mmr

// Peaks returns the array of mountain peaks in the MMR. This is completely
// deterministic given a valid mmr size. If the mmr size is invalid, this
// function returns nil.
//
// It is guaranteed that the peaks are listed in ascending order of position
// value.  The highest peak has the lowest position and is listed first. This is
// a consequence of the fact that the 'little' 'down range' peaks can only appear
// to the 'right' of the first perfect peak, and so on recursively.
//
// Note that as a matter of implementation convenience and efficiency the peaks
// are returned as *one based positions*
//
// So given the example below, which has an mmrSize of 17, the peaks are [15, 18]
//
//	3            15
//	           /    \
//	          /      \
//	         /        \
//	2       7          14
//	      /   \       /   \
//	1    3     6    10     13      18
//	    / \  /  \   / \   /  \    /  \
//	0  1   2 4   5 8   9 11   12 16   17
func Peaks(mmrSize uint64) []uint64 {
	if mmrSize == 0 {
		return nil
	}

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
		peakSize := TopPeak(mmrSize)

		// Because we *subtract* the computed peak size from mmrSize, we need to
		// recover the actual peak position. The arithmetic all works out so we
		// just accumulate the peakSizes as we go, and the result is always the
		// peak value against the original mmrSize we were given.
		peak = peak + peakSize
		peaks = append(peaks, peak)
		mmrSize -= peakSize
	}
	return peaks
}

func PeakHashes(store indexStoreGetter, mmrSize uint64) ([][]byte, error) {
	// Note: we can implement this directly any time we want, but lets re-use the testing for Peaks
	var path [][]byte
	for _, pos := range Peaks(mmrSize) {
		value, err := store.Get(pos - 1)
		if err != nil {
			return nil, err
		}
		path = append(path, value)
	}
	return path, nil
}

// TopPeak returns the smallest, leftmost, peak containing *or equal to* pos
//
// This is essentially a ^2 *floor* function for the accumulation of bits:
//
//	TopPeak(1) = TopPeak(2) = 1
//	TopPeak(2) = TopPeak(3) = TopPeak(4) = TopPeak(5) = TopPeak(6) = 3
//	TopPeak(7) = 7
//
//	2       7
//	      /   \
//	1    3     6    10
//	    / \  /  \   / \
//	0  1   2 4   5 8   9 11
func TopPeak(pos uint64) uint64 {

	// This works by working out the next peak up then subtracting 1, which is a flooring function for the bits over the current peak
	return 1<<(BitLength64(pos+1)-1) - 1
}

// PosFloor returns the index height and size of the largest perfect peak contained in, or exactly, pos
// This is essentially a ^2 *floor* function for the accumulation of bits:
//
//	PosFloor(1) = PosFloor(2) = 0, 1
//	PeakFloor(2) = PeakFloor(3) = PeakFloor(4) = PeakFloor(5) = PeakFloor(6) = 1, 3
//	PeakFloor(7) = 2, 7
//
//	2       7
//	      /   \
//	1    3     6    10
//	    / \  /  \   / \
//	0  1   2 4   5 8   9 11
func PosFloor(pos uint64) (uint64, uint64) {
	heightIndex := (BitLength64(pos+1) - 1)
	return heightIndex, 1<<heightIndex - 1
}

func HeightPeakRight(mmrSize uint64, height uint64, i uint64) (uint64, uint64, bool) {

	// jump to right sibling
	i += SiblingOffset(height)

	// then the left child
	for i > mmrSize-1 {
		if height == 0 {
			return 0, 0, false
		}
		height -= 1
		i -= (2 << height) // removes the parent offset
	}
	return height, i, true
}
