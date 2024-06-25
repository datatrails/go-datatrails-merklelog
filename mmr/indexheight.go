package mmr

// References:
// * https://github.com/proofchains/python-proofmarshal/blob/master/proofmarshal/mmr.py#L18
// * https://github.com/mimblewimble/grin/blob/0ff6763ee64e5a14e70ddd4642b99789a1648a32/core/src/core/pmmr.rs#L606

import (
	"math"
	"math/bits"
)

// Most of these methods mirror implementations in the references cited above

// JumpLeftPerfect is used to iteratively discover the left most node at the same
// height as the  node identified by pos. This is how we discover the height in
// the tree of an arbitrary position so as to avoid ever having to materialize
// the whole tree. It 'jumps left' by the size of the largest perfect tree which would
// precede pos.
//
// So given,
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
//
// JumpLeftPerfect(13) returns 6 because the size of the largest perfect tree
// preceding 13 is 7.  The next jump, JumpLeftPerfect(6) returns 3, because the
// perfect tree preceding 6 is size 3. and the 'all ones' node is found. And
// the count of 1's - 1 is the index height.
//
// ** Note ** that pos is the *one based* position not the zero based index.
func JumpLeftPerfect(pos uint64) uint64 {
	mostSignificantBit := uint64(1) << (BitLength64(pos) - 1)
	return pos - (mostSignificantBit - 1)
}

// IndexHeight obtains the tree height of an MMR index Taking advantage of the
// binary encoding resulting from the tree construction to do so. This function
// is the basis for the entire MMR implementation. See the extended remarks in
// doc.go for exposition on why & how it works.
func IndexHeight(i uint64) uint64 {
	// convert from zero based index to 1 based position, else the encoding doesn't work out
	return PosHeight(i + 1)
}

// MaxPeakHeight obtains the hight index of the highest (and left most peak)
// for the mmr index i
func MaxPeakHeight(i uint64) uint64 {

	height := uint64(bits.Len64(i+1)) - 1

	// edge case: if i represents a perfect peak, then we are done as node i is
	// included in the derived height.
	if AllOnes(i + 1) {
		return height
	}

	// otherwise, height is the height of the perfect tree that contains i, and
	// its position is *after* i. So the previous height is the highest peak
	// included in the mmr index i.
	return height - 1
}

// HeightIndexLeafCount returns the count of leaves that are contained in a single
// mountain whose height is heightIndex + 1
func HeightIndexLeafCount(heightIndex uint64) uint64 {

	// Given mountain height, the number of m nodes is:
	// 	m = (1 << h) - 1
	// The size can be computed from the number of leaves f as
	// 	m = f + f - 1
	// So to recover the number of f leaves in a single mountain:
	// 	f = (m + 1) / 2

	// m = (1 << h) - 1 accounting for it being an index
	m := HeightIndexSize(heightIndex)

	return (m + 1) / 2
}

// PosHeight is used when position is a 1 based count
func PosHeight(pos uint64) uint64 {
	for !AllOnes(pos) {
		pos = JumpLeftPerfect(pos)
	}
	return BitLength64(pos) - 1
}

// JumpRightSibling moves from pos to the next sibling at the same height
func JumpRightSibling(pos uint64) uint64 {
	return pos + (1 << (PosHeight(pos) + 1)) - 1
}

// LeftChild returns the position of the top most left child of parent pos.
// If pos is at height 0 it returns false (and true otherwise)
//
// So, some examples given in the diagram below:
//
//	pos 18 has height 1, and 18 - (1 << 1) =  18 - 2 = 16.
//	pos 14 has height 2, and 14 - (1 << 2) =  14 - 4 = 10.
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
func LeftChild(pos uint64) (uint64, bool) {
	height := PosHeight(pos)
	if height == 0 {
		return 0, false
	}
	return pos - (1 << height), true
}

// SiblingOffset returns the offset to the sibling at the given height.
func SiblingOffset(height uint64) uint64 {
	// for a 1 based height we would use (1 << height) - 1. this is the most
	// convenient form to take advantage of the 'all ones' property of the left
	// most peaks. but as our height is naturaly 0 based we start at 2 to
	// recover this.
	return (2 << height) - 1
}

func ParentOffset(height uint64) uint64 {
	// for a 1 based height we would use (1 << height) - 1, but as our height is 0 based we start at 2
	return 2 << height
}

// IndexHieght2 obtains the tree height of an MMR index (position - 1)
// Then, we iteratively reduce the size to the next perfect tree down. Each time
// we do this if pos sticks out, we shift it back under by shifting it according
// to the current tree size. This process is
// (this is the variant that most intuitively follows the typical construction diagrams, and is kept around for testing)
func IndexHeight2(pos uint64) uint64 {

	if pos == 0 {
		return 0
	}

	// Start with the smallest perfect peak that contains pos.  For any position
	// in the tree we know that the perfect tree containing it has all bits 1
	// and is the first such number greater than the position. Eg, counting from
	// 1, for position 4, the root node of the first perfectly balanced tree
	// that contains it, is at possition 7
	var peakSz uint64
	peakSz = math.MaxUint64 >> bits.LeadingZeros64(pos)

	for peakSz > 0 {
		// note: make the condition > if counting from 1
		if pos >= peakSz {
			pos -= peakSz
		}
		peakSz >>= 1
	}
	return pos
}
