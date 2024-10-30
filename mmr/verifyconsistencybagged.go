package mmr

import (
	"bytes"
	"hash"
)

// VerifyConsistencyBagged returns true if the mmr log update from mmr a to mmr b is
// append only.  This means that the new log contains an exact copy of the
// previous log, with any new nodes appended after. The proof is created by
// [datatrails/go-datatrails-merklelog/merklelog/mmr/IndexConsistencyProof]
//
// The proof comprises an single path which contains an inclusion proof for each
// peak node in the old mmr against the new mmr root. As all mmr interior nodes
// are committed to their mmr position when added, this is sufficient to show
// the new mmr contains an exact copy of the previous. And so can only be the
// result of append operations.
//
// There is, of course, some redundancy in the path, but accepting that allows
// re-use of VerifyInclusion for both consistency and inclusion proofs.
func VerifyConsistencyBagged(
	hasher hash.Hash, peakHashesA [][]byte,
	proof ConsistencyProof, rootA []byte, rootB []byte) bool {

	// There must be something to prove
	if len(peakHashesA) == 0 {
		return false
	}

	// Check the peakHashesA, which will have been retrieved from the updated
	// log, recreate rootA. rootA should have come from a previous Merkle
	// Signed Root.
	if !bytes.Equal(HashPeaksRHS(hasher, peakHashesA), rootA) {
		return false
	}

	// Catch the case where mmr b is exactly mmr a
	if len(proof.PathBagged) == 0 && bytes.Equal(rootA, rootB) {
		return true
	}

	// Otherwise, s zero length path not valid
	if len(proof.PathBagged) == 0 {
		return false
	}

	// Establish the node indices of the peaks in the original mmr A.  Those
	// peak nodes must be at the same indices in mmr B for the update to be
	// considered consistent. However, if mmr b has additional entries at all,
	// some or all of those peaks from A will no longer be peaks in B.
	peakPositions := PosPeaks(proof.MMRSizeA)

	var ok bool
	iPeakHashA := 0
	path := proof.PathBagged
	for ; iPeakHashA < len(peakHashesA); iPeakHashA++ {

		// Verify that the peak from A is included in mmr B. As the interior
		// node hashes commit the node position in the log, this can only
		// succeed if the peaks are both included and placed in the same
		// position.
		nodeHash := peakHashesA[iPeakHashA]

		var proofLen int

		ok, proofLen = VerifyFirstInclusionPathBagged(
			proof.MMRSizeB, hasher, nodeHash, peakPositions[iPeakHashA]-1,
			path, rootB)
		if !ok || proofLen > len(path) {
			return false
		}
		path = path[proofLen:]
	}

	// Note: only return true if we have verified the complete path.
	return ok && len(path) == 0
}

// CheckConsistencyBagged is used to check that a new log update is consistent With
// respect to some previously known "bagged" root and the current store.
func CheckConsistencyBagged(
	store indexStoreGetter, hasher hash.Hash,
	cp ConsistencyProof, rootA []byte) (bool, []byte, error) {

	iPeaks := PosPeaks(cp.MMRSizeA)

	// logger.Sugar.Infof(".... PeakBagRHS: %v", iPeaks)
	peakHashesA, err := PeakBagRHS(store, hasher, 0, iPeaks)
	if err != nil {
		return false, nil, err
	}

	// logger.Sugar.Infof(".... GetRoot")
	rootB, err := GetRoot(cp.MMRSizeB, store, hasher)
	if err != nil {
		return false, nil, err
	}

	return VerifyConsistencyBagged(
		hasher, peakHashesA, cp, rootA, rootB), rootB, nil
}
