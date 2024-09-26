package mmr

import "hash"


// IncludedRoot calculates the accumulator peak for the provided
// proof and node value. Note that both interior and leaf nodes are handled
// identically
//
// Arguments:
// 	* i is the index the nodeHash is to be shown at
// 	* nodehash the value whose inclusion is to be shown
//  * proof is the path of ibling values committing i. They recreate the unique
//  accumulator peak that committed i to the MMR state from which the proof was
//  produced.
func IncludedRoot(hasher hash.Hash, i uint64, nodeHash []byte, proof [][]byte) []byte {

	root := nodeHash

	g := IndexHeight(i)

	for _, sibling := range proof {

		// If the index after i is higher, it is the left parent,
		// and i is the right sibling

		if IndexHeight(i+1) > g {

			// The parent of a right sibling is stored immediately after
			i = i + 1

			// Set `root` to `H(i+1 || sibling || root)`
			root = HashPosPair64(hasher, i+1, sibling, root)
		} else {

			// The parent of a left sibling is stored immediately after
			// its right sibling.

			i = i + (2 << g)

			//  Set `root` to `H(i+1 || root || sibling)`
			root = HashPosPair64(hasher, i+1, root, sibling)
		}

		// Set g to the height of the next item in the path.
		g = g + 1
	}

	return root
}
