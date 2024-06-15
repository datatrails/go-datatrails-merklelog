package mmr

import (
	"errors"
)

var (
	ErrProofLenTooLarge = errors.New("proof length value is too large")
	ErrPeakListTooShort = errors.New("the list of peak values is too short")
)

// GetProofPeakRoot returns the peak hash for sub tree containing the node
// index.  This is a convenience method to assist with general proof
// verification. In many contexts, where the leaf count is known, or if the
// index is known to represent a leaf, it is more efficient and clearer to do
// this directly.
//
// A proof for node 2 would be [5] and the peak list for mmrSize 11 would be
//
//	[6, 9, 10]
//
// To obtain the appropriate root to verify a proof of inclusion for node two call this function with:
//
//	peakHashes: [H(6), H(9), H(10)]
//	proofLen: 1
//	mmrSize: 11
//	mmrIndex: 2
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
func GetProofPeakRoot(peakHashes [][]byte, proofLen int, mmrSize, mmrIndex uint64) ([]byte, error) {

	// for leaf nodes, the peak height index is the proof length - 1, for
	// generality, to account for interior nodes, we use IndexHeight here.
	// In contexts where consistency proofs are being generated to check log
	// extension, typically the returned height from IndexProofPath is
	// available.
	peakIndex := GetProofPeakIndex(proofLen, mmrSize, IndexHeight(mmrIndex))
	if peakIndex >= len(peakHashes) {
		return nil, ErrPeakListTooShort
	}
	return peakHashes[peakIndex], nil
}

// GetLeafProofRoot gets the appropriate peak root from peakHashes for a leaf proof, See GetProofPeakRoot
func GetLeafProofRoot(peakHashes [][]byte, proof [][]byte, mmrSize uint64) ([]byte, error) {
	peakIndex := GetProofPeakIndex(len(proof), mmrSize, 0)
	if peakIndex >= len(peakHashes) {
		return nil, ErrPeakListTooShort
	}
	return peakHashes[peakIndex], nil
}

// GetLeafProofRoot gets the compressed accumulator peak index for a leaf proof, See GetProofPeakRoot
func GetProofPeakIndex(proofLen int, mmrSize uint64, heightIndex uint64) int {
	peakHeightIndex := uint64(proofLen) + heightIndex

	// get the index into the accumulator
	// peakMap is also the leaf count, which is often known to the caller
	peakMap := PeaksBitmap(mmrSize)
	return PeakIndex(peakMap, peakHeightIndex)
}

// IndexProofPath collects the merkle root proof for the local MMR peak containing index i
//
// So for the following index tree, and i=15 with mmrSize = 26 we would obtain the path
//
// [H(16), H(20)]
//
// Because the local peak is 21, and given the value for 15, we only need 16 and
// then 20 to prove the local root.
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
func IndexProofPath(mmrSize uint64, store indexStoreGetter, i uint64) ([][]byte, uint64, uint64, error) {

	var iSibling uint64
	var iLocalPeak uint64

	var proof [][]byte
	heightIndex := IndexHeight(i) // allows for proofs of interior nodes

	for { // iSibling is guaranteed to break the loop

		iLocalPeak = i

		if IndexHeight(i+1) > heightIndex {
			iSibling = i - SiblingOffset(heightIndex)
			i += 1 // move i to parent
		} else {
			iSibling = i + SiblingOffset(heightIndex)
			i += 2 << heightIndex // move i to parent
		}

		if iSibling >= mmrSize {
			return proof, iLocalPeak, heightIndex, nil
		}

		value, err := store.Get(iSibling)
		if err != nil {
			return nil, 0, heightIndex, err
		}
		proof = append(proof, value)

		heightIndex += 1
	}
}

// IndexProof is a convenience wrapper for IndexProofPath
// For circumstances where the peak index and the peak heighIndex are not required by the caller
func IndexProof(mmrSize uint64, store indexStoreGetter, i uint64) ([][]byte, error) {
	proof, _, _, err := IndexProofPath(mmrSize, store, i)
	return proof, err
}

// LeftPosForHeight returns the position that is 'most left' for the given height.
// Eg for height 0, it returns 0, for height 1 it returns 2, for 2 it returns 6.
// Note that these are always values where the corresponding 1 based position
// has 'all ones' set.
func LeftPosForHeight(height uint64) uint64 {
	return (1 << (height + 1)) - 2
}
