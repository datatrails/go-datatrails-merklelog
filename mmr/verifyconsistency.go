package mmr

import (
	"hash"
)

func CheckConsistency(
	store indexStoreGetter, hasher hash.Hash,
	cp ConsistencyProof, peakHashesA [][]byte) (bool, error) {

	peakHashesB, err := PeakHashes(store, cp.MMRSizeB)
	if err != nil {
		return false, err
	}

	return VerifyConsistency(
		hasher, cp, peakHashesA, peakHashesB), nil
}

// VerifyConsistency verifies the consistency between two MMR states.
// The MMR(A) and MMR(B) states are identified by the fields MMRSizeA and
// MMRSizeB in the proof. peakHashesA and B are the node values corresponding to
// the MMR peaks of each respective state. The Path in the proof contains the
// nodes necessary to prove each A-peak reaches a B-peak. The path contains the
// concatenated inclusion proofs for each A-peak in MMR(B).
//
//	    MMR(A):[7, 8]      MMR(B):[7, 10, 11]
//	 2       7                7
//	       /   \            /   \
//	 1    3     6          3     6    10
//	     / \  /  \        / \  /  \   / \
//	 0  1   2 4   5 8    1   2 4   5 8   9 11
//
//		Path MMR(A) -> MMR(B)
//		7 in MMR(B) -> []
//		7 in MMR(B) -> [9]
//		Path = [9]
//
// Noting that the path is empty for node 7 and catenate([], [9]) = [9]
func VerifyConsistency(
	hasher hash.Hash,
	proof ConsistencyProof, peakHashesA [][]byte, peakHashesB [][]byte) bool {

	// Establish the node indices of the peaks in each mmr state.  The peak
	// nodes of mmr state A must be at the same indices in mmr B for the update
	// to be considered consistent. However, if mmr b has additional entries at
	// all, some or all of those peaks from A will no longer be peaks in B.
	peakPositionsA := Peaks(proof.MMRSizeA)
	peakPositionsB := Peaks(proof.MMRSizeB)

	// Require the peak hash list length to match the number of peaks in the mmr
	// state identified by the MMRSize's.
	// This also catches the various corner cases where the hashes are incorrect lengths.
	if len(peakHashesA) != len(peakPositionsA) {
		return false
	}
	if len(peakHashesB) != len(peakPositionsB) {
		return false
	}

	var ok bool
	var proofLen int
	iPeakA := 0
	iPeakB := 0
	path := proof.Path

	posA := peakPositionsA[iPeakA] // pos because it may no longer be a peak in MMR(B)
	for iPeakA < len(peakHashesA) {

		// Each a-peak in A will have as its root, the *first* b-peak in B whose
		// position is >= the a-peak position. We may not, and typically will
		// not, consume all the peaks in MMR(B).
		// Note that where the a-peak is also present as a b-peak, the path will
		// be empty and that VerifyInclusionPath deals with that case by
		// requiring the provided leaf matches the provided root. The positions
		// are naturally equal in both in this case.

		peakB := peakPositionsB[iPeakB] // peak because it is a peak in MMR(B)
		for posA <= peakB {
			ok, proofLen = VerifyInclusionPath(
				proof.MMRSizeB, hasher, peakHashesA[iPeakA], posA-1,
				path, peakHashesB[iPeakB])
			if !ok || proofLen > len(path) {
				return false
			}
			// proofLen will be 0 in the case where posA == peakB,  and this works for the case where MMR(A) == MMR(B)
			path = path[proofLen:]
			iPeakA++
			if iPeakA == len(peakHashesA) {
				break
			}
			posA = peakPositionsA[iPeakA]
		}
		iPeakB++
	}

	// Note: only return true if we have verified the complete path.
	return ok && len(path) == 0
}
