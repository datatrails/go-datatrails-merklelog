package mmr

import "hash"

// ConsistencyProof describes a proof that the merkle log defined by size a is
// perfectly contained in the log described by size b. This structure aligns us
// with the consistency proof format described in this ietf draft:
// https://datatracker.ietf.org/doc/draft-ietf-cose-merkle-tree-proofs/ The
// proof should be verified against a previously signed root for the mmr size a
// and the root of the proposed log defined by size b.
//
// A reference introducing the concept of consistency proofs in merkle trees:
// https://pangea.cloud/docs/audit/merkle-trees#outline-consistency-proof
type ConsistencyProof struct {
	MMRSizeA uint64
	MMRSizeB uint64
	Path     [][]byte
}

// IndexConsistencyProof creates a proof that mmr B appends to mmr A.
// Our method works by generating inclusion proofs for each of the peaks of A.
// This method results in a proof path that has some redundancy in it, but
// permits re-use of the inclusion proof verification method.
//
// As each peak is an interior node, and as each interior node commits to the
// number of nodes under it (the count of nodes at that point) there is only one
// possible location the node can exist in the tree. If node x is in both mmr A
// and mmr B then it is included in exactly the same position.
//
// Verification will first show that the root of A can be re-produced from MMR
// B, and then proceed to checking the inclusion proofs for the A peaks in mmr
// B.
func IndexConsistencyProof(
	mmrSizeA, mmrSizeB uint64, store indexStoreGetter, hasher hash.Hash,
) (ConsistencyProof, error) {

	proof := ConsistencyProof{
		MMRSizeA: mmrSizeA,
		MMRSizeB: mmrSizeB,
	}

	// Find the peaks corresponding to the previous mmr
	peaksA := Peaks(mmrSizeA)

	// Now generate peak proofs against the new mmr size, using the peak indices
	// as the input indices to prove
	for _, iPeakA := range peaksA {
		peakProof, err := IndexProof(mmrSizeB, store, hasher, iPeakA-1)
		if err != nil {
			return ConsistencyProof{}, err
		}
		proof.Path = append(proof.Path, peakProof...)
	}
	return proof, nil
}
