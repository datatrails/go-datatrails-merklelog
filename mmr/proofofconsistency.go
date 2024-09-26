package mmr

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
	MMRSizeA uint64 `cbor:"1,keyasint"`
	MMRSizeB uint64 `cbor:"2,keyasint"`
	// legacy proof format
	PathBagged [][]byte   `cbor:"3,keyasint"`
	Path       [][][]byte `cbor:"4,keyasint"`
}

// IndexConsistencyProof creates a proof that mmr B appends to mmr A.
// Our method works by generating inclusion proofs for each of the peaks of A.
//
// As each peak is an interior node, and as each interior node commits to the
// number of nodes under it (the count of nodes at that point) there is only one
// possible location the node can exist in the tree. If node x is in both mmr A
// and mmr B then it is included in exactly the same position.
//
// Verification is then performed in terms of the mmr accumulator states MMR(A)
// and MMR(B) for each "old" peak in MMR(A) we show there is a path to a "new"
// or "same" peak in MMR(B)
func IndexConsistencyProof(
	store indexStoreGetter, mmrSizeA, mmrSizeB uint64,
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

		peakProof, err := IndexProof(store, mmrSizeB, iPeakA-1)
		if err != nil {
			return ConsistencyProof{}, err
		}
		proof.Path = append(proof.Path, peakProof)
	}
	return proof, nil

}
