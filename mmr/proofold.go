package mmr

// Note: the expectation is that once we are satisfied with the new methods we
// will delete this file A reason to keep it around is that testing may benefit
// from having multiple implementations of key algorithms

// InclusionProofLocalOld is depreciated and retained only for testing
// See InclusionProofLocal instead
//
// collects the merkle root proof for the local MMR peak containing index i
//
// So for the follwing index tree, and i=15 with mmrSize = 26 we would obtain the path
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
func InclusionProofLocalOld(mmrSize uint64, store indexStoreGetter, i uint64) ([][]byte, uint64, error) {

	var proof [][]byte
	height := IndexHeight(i) // allows for proofs of interior nodes

	var err error
	var value []byte

	for i < mmrSize {
		iHeight := IndexHeight(i)
		iNextHeight := IndexHeight(i + 1)
		if iNextHeight > iHeight {
			iSibling := i - SiblingOffset(height)
			if iSibling >= mmrSize {
				break
			}

			if value, err = store.Get(iSibling); err != nil {
				return nil, 0, err
			}
			proof = append(proof, value)
			// go to parent node
			i += 1
		} else {
			iSibling := i + SiblingOffset(height)
			if iSibling >= mmrSize {
				break
			}

			if value, err = store.Get(iSibling); err != nil {
				return nil, 0, err
			}
			proof = append(proof, value)
			// goto parent node
			i += 2 << height
		}
		height += 1
	}
	return proof, i, nil
}
