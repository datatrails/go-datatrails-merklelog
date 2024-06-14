package mmr

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
