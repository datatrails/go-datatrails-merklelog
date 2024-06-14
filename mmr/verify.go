package mmr

import (
	"bytes"
	"hash"
)

// VerifyInclusionPath returns true if the leafHash combined with path, reproduces the provided root
//
// To facilitate the concatenated proof paths used for consistency proofs, it
// returns the count of path elements used to reach the root.
//
// root: The local "peak" root in which leafHash is recorded. This root is a
// member of the current mmr accumulator, or is itself a node which can be verified
// for inclusion in a future accumulator.
func VerifyInclusionPath(
	mmrSize uint64, hasher hash.Hash, leafHash []byte, iNode uint64, proof [][]byte, root []byte,
) (bool, int) {

	// Deal with the degenerate case where iNode is a perfect peak. The proof will be nil.
	if len(proof) == 0 && bytes.Equal(leafHash, root) {
		return true, 0
	}

	pos := iNode + 1
	heightIndex := PosHeight(pos) // allows for proofs of interior nodes
	elementHash := leafHash

	for iProof, p := range proof {

		hasher.Reset()

		// If the next node is higher, are at the right child, and the left otherwise
		if PosHeight(pos+1) > heightIndex {
			// we are at the right child

			pos += 1
			HashWriteUint64(hasher, pos) // pos is now the parent pos, which was also the commit value
			hasher.Write(p)
			hasher.Write(elementHash)
		} else {
			// we are at the left child

			pos += 2 << heightIndex
			HashWriteUint64(hasher, pos) // pos is now the parent pos, which was also the commit value
			hasher.Write(elementHash)
			hasher.Write(p)
		}

		elementHash = hasher.Sum(nil)

		if bytes.Equal(elementHash, root) {
			// If we have the root then we have successfully completed the
			// current proof.  Return the index for the start of the next
			return true, iProof + 1
		}

		heightIndex += 1
	}
	return false, len(proof)
}
