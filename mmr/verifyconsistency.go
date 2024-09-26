package mmr

import (
	"bytes"
	"errors"
	"hash"
)

var (
	ErrConsistencyCheck = errors.New("consistency check failed")
)

// CheckConsistency verifies that the current state mmrSizeB is consistent with
// the provided accumulator for the earlier size A The provided accumulator
// (peakHashesA) should be taken from a trusted source, typically a signed mmr
// state.
//
// See VerifyConsistency for more.
func CheckConsistency(
	store indexStoreGetter, hasher hash.Hash,
	mmrSizeA, mmrSizeB uint64, peakHashesA [][]byte) (bool, [][]byte, error) {

	// Obtain the proofs from the current store
	cp, err := IndexConsistencyProof(store, mmrSizeA, mmrSizeB)
	if err != nil {
		return false, nil, err
	}

	// Obtain the expected resulting peaks from the current store
	peakHashesB, err := PeakHashes(store, cp.MMRSizeB)
	if err != nil {
		return false, nil, err
	}

	return VerifyConsistency(hasher, cp, peakHashesA, peakHashesB)
}

// VerifyConsistency verifies the consistency between two MMR states.
//
// The MMR(A) and MMR(B) states are identified by the fields MMRSizeA and
// MMRSizeB in the proof. peakHashesA and B are the node values corresponding to
// the MMR peaks of each respective state. The Path in the proof contains the
// nodes necessary to prove each A-peak reaches a B-peak. The path contains the
// inclusion proofs for each A-peak in MMR(B).
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
//		8 in MMR(B) -> [9]
//		Path = [[], [9]]
func VerifyConsistency(
	hasher hash.Hash,
	cp ConsistencyProof, peaksFrom [][]byte, peaksTo [][]byte) (bool, [][]byte, error) {

	// Get the peaks proven by the consistency proof using the provided peaks
	// for mmr size A
	proven, err := ConsistentRoots(hasher, cp.MMRSizeA, peaksFrom, cp.Path)
	if err != nil {
		return false, nil, err
	}

	// If all proven nodes match an accumulator peak for MMR(sizeB) then MMR(sizeA)
	// is consistent with MMR(sizeB). Because both the peaks and the accumulator
	// peaks are listed in descending order of height this can be accomplished
	// with a linear scan.

	ito := 0
	for _, root := range proven {

		if bytes.Equal(peaksTo[ito], root) {
			continue
		}

		// If the root does not match the current peak then it must match the
		// next one down.

		ito += 1

		if ito >= len(peaksTo) {
			return false, nil, ErrConsistencyCheck
		}

		if !bytes.Equal(peaksTo[ito], root) {
			return false, nil, ErrConsistencyCheck
		}
	}

	// All proven peaks have been matched against the future accumulator. The log
	// committed by the future accumulator is consistent with the previously
	// committed log state.
	return true, proven, nil
}
