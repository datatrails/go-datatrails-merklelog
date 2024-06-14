package mmr

// Note: the expectation is that once we are satisfied with the new methods we
// will delete this file A reason to keep it around is that testing may benefit
// from having multiple implementations of key algorithms

// PeaksOld is deprecated and retained only for reference and testing.
//
// returns the array of mountain peaks in the MMR. This is completely
// deterministic given a valid mmr size. If the mmr size is invalid, this
// function returns nil.
//
// It is guaranteed that the peaks are listed in ascending order of position
// value.  The highest peak has the lowest position and is listed first. This is
// a consequence of the fact that the 'little' 'down range' peaks can only appear
// to the 'right' of the first perfect peak, and so on recursively.
//
// Note that as a matter of implementation convenience and efficency the peaks
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
func PeaksOld(mmrSize uint64) []uint64 {
	if mmrSize == 0 {
		return nil
	}

	// catch invalid range, where siblings exist but no parent exists
	if PosHeight(mmrSize+1) > PosHeight(mmrSize) {
		return nil
	}

	// The top peak is always the left most and, when counting from 1, will have all binary '1's
	top := TopPeak(mmrSize)

	peaks := []uint64{top}
	peak := top
OuterLoop:
	for {
		peak = JumpRightSibling(peak)
		for peak > mmrSize {
			if p, ok := LeftChild(peak); ok {
				peak = p
				continue
			}
			break OuterLoop
		}
		peaks = append(peaks, peak)
	}
	return peaks
}
