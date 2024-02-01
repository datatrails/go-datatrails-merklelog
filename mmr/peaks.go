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
func Peaks(mmrSize uint64) []uint64 {
	if mmrSize == 0 {
		return nil
	}

	// catch invalid range, where siblings exist but no parent exists
	if PosHeight(mmrSize+1) > PosHeight(mmrSize) {
		return nil
	}

	// The top peak is always the left most and, when counting from 1, will have all binary '1's
	top := uint64(1)
	for (top - 1) <= mmrSize {
		top <<= 1
	}
	top = (top >> 1) - 1
	if top == 0 {
		return nil
	}

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

// HighestPos returns the height and the peak index for the highest and
// most left node in the MMR of the given size.
func HighestPos(mmrSize uint64) (uint64, uint64) {
	height := uint64(0)
	iPrev := uint64(0)
	i := LeftPosForHeight(height)
	for i < mmrSize {
		height += 1
		iPrev = i
		i = LeftPosForHeight(height)
	}
	return height - 1, iPrev
}
