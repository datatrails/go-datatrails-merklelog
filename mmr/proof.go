package mmr

import (
	"errors"
)

var (
	ErrProofLenTooLarge = errors.New("proof length value is too large")
	ErrPeakListTooShort = errors.New("the list of peak values is too short")
)

// GetProofPeakRoot returns the peak hash for sub tree committing any node.
//
// This is a convenience for use when the caller does not have the heightIndex naturaly from other operations.
//
// A proof for node 2 would be [5], and the peak list for mmrSize 11 would be
//
//	[6, 9, 10]
//
// To obtain the appropriate root to verify a proof of inclusion for node 2 call this function with:
//
//	peakHashes: [H(6), H(9), H(10)]
//	proofLen: 1
//	mmrSize: 11
//	heightIndex: 1
//
// The returned peak root will be H(6)
//
// For node 7, mmrIndex would be 7, all other parameters would remain the same and the returned value would be H(9)
//
//	2        6
//	       /   \
//	1     2     5      9
//	     / \   / \    / \
//	0   0   1 3   4  7   8 10
func GetProofPeakRoot(mmrSize uint64, mmrIndex uint64, peakHashes [][]byte, proofLen int) ([]byte, error) {

	// for leaf nodes, the peak height index is the proof length - 1, for
	// generality, to account for interior nodes, we use IndexHeight here.
	// In contexts where consistency proofs are being generated to check log
	// extension, typically the returned height from InclusionProofPath is
	// available.

	heightIndex := IndexHeight(mmrIndex)

	peakIndex := GetProofPeakIndex(mmrSize, proofLen, uint8(heightIndex))
	if peakIndex >= len(peakHashes) {
		return nil, ErrPeakListTooShort
	}
	return peakHashes[peakIndex], nil
}

// GetLeafProofRoot gets the appropriate peak root from peakHashes for a leaf proof, See GetProofPeakRoot
func GetLeafProofRoot(peakHashes [][]byte, proof [][]byte, mmrSize uint64) ([]byte, error) {
	peakIndex := GetProofPeakIndex(mmrSize, len(proof), 0)
	if peakIndex >= len(peakHashes) {
		return nil, ErrPeakListTooShort
	}
	return peakHashes[len(peakHashes)-peakIndex-1], nil
}

// GetLeafProofRoot gets the compressed accumulator peak index for a leaf proof, See GetProofPeakRoot
func GetProofPeakIndex(mmrSize uint64, d int, heightIndex uint8) int {
	// get the index into the accumulator
	// peakMap is also the leaf count, which is often known to the caller
	peakMap := PeaksBitmap(mmrSize)
	return PeakIndex(peakMap, int(heightIndex)+d)
}

// IndexPath collects the merkle proof mmr index i
//
// For the following index tree, and i=15 with mmrSize = 26 we would obtain the path
//
// [H(16), H(20)]
//
// Because the accumulator peak committing 15 is 21, and given the value for 15, we only need 16 and
// then 20 to verify the proof.
//
//	3              14
//	             /    \
//	            /      \
//	           /        \
//	          /          \
//	2        6            13           21
//	       /   \        /    \
//	1     2     5      9     12     17     20     24
//	     / \   / \    / \   /  \   /  \
//	0   0   1 3   4  7   8 10  11 15  16 18  19 22  23   25
func InclusionProof(store indexStoreGetter, mmrLastIndex uint64, i uint64) ([][]byte, error) {

	var iSibling uint64

	var proof [][]byte
	g := IndexHeight(i) // allows for proofs of interior nodes

	for { // iSibling is guaranteed to break the loop

		// The sibling of i is at i +/- 2^(g+1)
		siblingOffset := uint64((2 << g))

		// If the index after i is heigher, it is the left parent, and i is the right sibling.
		if IndexHeight(i+1) > g {
			// The witness to the right sibling is offset behind i
			iSibling = i - siblingOffset + 1

			// The parent of a right sibling is stored imediately after the sibling
			i += 1
		} else {

			// The witness to the left sibling is offset ahead of i
			iSibling = i + siblingOffset - 1

			// The parent of a left sibling is stored imediately after its right sibling
			i += siblingOffset
		}

		// When the computed sibling exceedes the range of MMR(C+1),
		// we have completed the path.
		if iSibling > mmrLastIndex {
			return proof, nil
		}

		value, err := store.Get(iSibling)
		if err != nil {
			return nil, err
		}
		proof = append(proof, value)

		// Set g to the height of the next item in the path.
		g += 1
	}
}

//	returns the mmr indices identifying the witness nodes for mmr index i
//
// This method allows tooling to individually audit the proof path node values for a given index.
func InclusionProofPath(mmrLastIndex uint64, i uint64) ([]uint64, error) {

	var iSibling uint64

	var proof []uint64
	g := IndexHeight(i) // allows for proofs of interior nodes

	for { // iSibling is guaranteed to break the loop

		// The sibling of i is at i +/- 2^(g+1)
		siblingOffset := uint64((2 << g))

		// If the index after i is heigher, it is the left parent, and i is the right sibling.
		if IndexHeight(i+1) > g {
			// The witness to the right sibling is offset behind i
			iSibling = i - siblingOffset + 1

			// The parent of a right sibling is stored imediately after the sibling
			i += 1
		} else {

			// The witness to the left sibling is offset ahead of i
			iSibling = i + siblingOffset - 1

			// The parent of a left sibling is stored imediately after its right sibling
			i += siblingOffset
		}

		// When the computed sibling exceedes the range of MMR(C+1),
		// we have completed the path.
		if iSibling > mmrLastIndex {
			return proof, nil
		}

		proof = append(proof, iSibling)

		// Set g to the height of the next item in the path.
		g += 1
	}
}

// LeftPosForHeight returns the position that is 'most left' for the given height.
// Eg for height 0, it returns 0, for height 1 it returns 2, for 2 it returns 6.
// Note that these are always values where the corresponding 1 based position
// has 'all ones' set.
func LeftPosForHeight(height uint64) uint64 {
	return (1 << (height + 1)) - 2
}
