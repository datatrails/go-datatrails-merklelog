package mmr

import "hash"

// Note: the expectation is that once we are satisfied with the new methods we
// will delete this file A reason to keep it around is that testing may benefit
// from having multiple implementations of key algorithms

// VerifyInclusionOld returns true if the provided proof demonstrates inclusion of
// nodeHash at position iLeaf+1
//
// proof and root should be obtained via IndexProof and GetRoot respectively.
//
// Remembering that the proof layout is this:
//
//	[local-peak-proof-i, right-sibling-of-i, left-of-i-peaks-reversed]
//
// And given the following MMR
//
//	3            15
//	           /    \
//	          /      \
//	         /        \
//	2       7          14             22
//	      /   \       /   \          /   \
//	1    3     6    10     13      18      21      25
//	    / \  /  \   / \   /  \    /  \    /  \    /   \
//	0  1   2 4   5 8   9 11   12 16   17 19   20 23   24 26
//
// Note that only the local-peak-proof-i elements will include the commitment to
// the number of descendent tree nodes. This means we must include H(pos) for
// each step in local-peak-proof-i, but then exclude it in all the others.
//
// So if we have a proof for leaf position 17 (iLeaf=16) the proof will be
// composed of the local peak proof for 17, which is
//
//	[ValueAt(16), ValueAt(21), Bagged-Peaks-RHS, Reveresed-LHS-Peaks]
//
// To correctly account for the position in the proof, we need to pre-pend the
// position for each element in the local peak proof:
//
// H(22 | V(21) | H(18|leaf|V(16)))
//
// Remembering that, confusingly, we always include the value for the 'right'
// node first despite the fact that reading order makes this seem 'on the left'
func VerifyInclusionOld(
	mmrSize uint64, hasher hash.Hash, nodeHash []byte, iNode uint64, proof [][]byte, root []byte,
) bool {
	ok, proofLen := VerifyFirstInclusionPathBagged(mmrSize, hasher, nodeHash, iNode, proof, root)
	return ok && proofLen == len(proof)
}
