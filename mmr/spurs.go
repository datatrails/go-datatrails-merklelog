package mmr

import "math/bits"

// ## Tree Spurs
//
// When dealing with 'chunks' of mmrs, which we call massifs, it is often
// necessary to know the number of 'extra' nodes in the chunk which over hang
// (and are dependent on) nodes from earlier massifs. When dealing directly with
// leaf indices (as opposed to tree indices), we often want to know the number
// of interior nodes 'above and to the left' of a specific node. As this is the
// count of interior nodes that are required to include that particular leaf in
// the tree. We call these slices through the tree 'spurs'. When we are dealing
// with massif chunks, we are really just treating the whole massif as a single
// leaf in a smaller mmr so the arithmetic is all the same. This allows us to
// determine the minimum necessary book keeping to keep each massif chunk self
// contained.

// SpurSumHeight counts the interior 'spur' nodes required for the given height
// The height is typically relative to the massif height.
//
// Note that the count *including* the last spur is obtained by adding the
// argument height to the result
// See doc.go SpurSumHeight for ascii art and extended explanation
//
// Each round i, starting at 1, calculates the *number* of spurs with height i,
// and multiplies by the length of that spur. the length of a spur is also its
// height which is also i.
//
// This technique also forms the basis of a fairly efficient, O log base 2 n
// ish, method to obtain the tree index from a leaf index.
//
// Mathjax:
//
//	\(sum = {\sum_{i=1}^{h-1}} 2^{h-1}/2^{i} * i\)
//
// => \(sum = {\sum_{i=1}^{h-1}} 2^{h-1-i} * i\)
//
// And as these are all power 2 operations, we can just shift
func SpurSumHeight(height uint64) uint64 {

	if height == 0 {
		return 0
	}

	// Each round i, starting at 1, calculates the *number* of spurs with height i,
	// and multiplies by the length of that spur. the length of a spur is also its
	// height which is also i.

	sum := uint64(0)
	for i := uint64(1); i <= height-1; i += 1 {
		sum += (1 << (height - 1 - i)) * i
	}
	return sum
}

// LeafMinusSpurSum returns the number of peaks preceding iLeaf that the future tree requires.
//
// This is simply the leaf index *minus* the count of nodes cast 'into shade' by
// the accumulated interior nodes preceding iLeaf.
//
// This corresponds to the number of preceding nodes that will be required to
// derive future interior nodes. If those preceding nodes are maintained in a
// stack, this is the current length of the stack.
//
// considering the following mmr
//
//	3            14                             29
//	           /    \                                \
//	          /      \                     /          \
//	         /        \                   /            \
//	2      6 .      .  13                21            28
//	      /   \       /   \             / . \         /   \
//	1    2     5     9     12         17     20     24     27
//	    / \  /  \   / \    /  \      /  \   / \     / \ . ./ \
//	   0   1 3   4  7   8 10   11  15   16 18  19  22  23 25  26 MMR INDICES
//	   0 . 1 2 . 3  4   5  6    7 . 8 .  9 10  11  12  13 14  15 LEAF INDICES
//
// We start by adding all 11, which as we are zero based is just 11 directly.
// Then there are (11 - 1) / 2 spurs which are _at least_ 1 long
// There are a remaining 5 / 2 which are at _at least_ 2 long
// Then finally there is 2 /2 which is _at least_ 3 long
// All the 'odd' leafs have 'zero' length spurs
//
// So we are accounting for all the spurs in parallel, and reducing the set of
// spurs in play on each round. This means the 'count' to subtract is exactly
// the number of spurs remaining in the set (ie we are always 1 x set length).
// Due to the binary nature of the tree, the set reduction is just dividing the
// current number of spurs by 2 and the count to subtract is exactly the result
// of that.
func LeafMinusSpurSum(leafIndex uint64) uint64 {

	// XXX: TODO: I think there is a more efficient approach which recursively
	// splits the leaf index into perfect sub trees based on the most sig bit
	// set, and then uses sum = 2i at each round. But this approach, especially
	// given it is used mostly for the much smaller logical massif connecting
	// tree, is fine too.

	sum := leafIndex
	leafIndex >>= 1
	for ; leafIndex > 0; leafIndex >>= 1 {
		sum -= leafIndex
	}
	return sum
}

// SpurHeightLeaf returns the number of nodes 'above' and to the *left* of the provided leaf index
//
// Notice this is the leaf index as tho the leaves were in their own array
// rather than the mmr index
//
// considering the following mmr
//
//	3            14                             29
//	           /    \                                \
//	          /      \                     /          \
//	         /        \                   /            \
//	2      6 .      .  13                21            28
//	      /   \       /   \             / . \         /   \
//	1    2     5     9     12         17     20     24     27
//	    / \  /  \   / \    /  \      /  \   / \     / \ . ./ \
//	   0   1 3   4  7   8 10   11  15   16 18  19  22  23 25  26 MMR INDICES
//	   0 . 1 2 . 3  4   5  6    7 . 8 .  9 10  11  12  13 14  15 LEAF INDICES
//
// iLeaf = 3 returns 2, iLeaf 7 returns 3, iLeaf 9 returns 1
// Notice that all the even numbered iLeaf, eg 2, 4, 6, 8  all return 0,
func SpurHeightLeaf(leafIndex uint64) uint64 {
	// The binary tree structure means we can use the count of least significant
	// zero bits as a proxy for height
	return uint64(bits.TrailingZeros64(leafIndex + 1))
}

// TreeIndex returns the mmr index of the i'th leaf It can also be used to
// calculate the sum of all the 'alpine nodes' in the mmr blobs preceding the
// blob if the blob index is substituted for iLeaf
func TreeIndexOld(leafIndex uint64) uint64 {

	// XXX: TODO it feels like there is a way to initialize using SpurSumHeight
	// then accumulate using some variation of the inner term of SpurSumHeight.
	// But the approach is already O(Log 2 n) ish.

	sum := uint64(0)
	for i := leafIndex; i > 0; {
		height := Log2Uint64(i) + 1
		sum += SpurSumHeight(height) + height
		half := uint64(1 << (height - 1))
		i -= half
	}
	return sum
}
