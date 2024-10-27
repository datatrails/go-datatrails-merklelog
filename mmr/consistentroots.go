package mmr

import (
	"bytes"
	"errors"
	"hash"
)

var (
	ErrAccumulatorProofLen = errors.New("a proof for each accumulator is required")
)

// ConsistentRoots  is supplied with the accumulator from which consistency is
// being shown, and an inclusion proof for each accumulator entry in a future MMR
// state.
//
// The algorithm recovers the necessary prefix (peaks) of the future
// accumulator against which the proofs were obtained.
// It is typical that many nodes in the original accumulator share the same peak in the new accumulator.
// The returned list will be a descending height ordered list of elements from the
// accumulator for the consistent future state. It may be exactly the future
// accumulator or it may be a prefix of it.
//
// The order of the roots returned matches the order of the nodes in the accumulator.
//
// Args:
//   - fromSize the size the complete MMR from which consistency was proven.
//   - accumulatorfrom the node values correponding to the peaks of the accumulator at MMR(sizeA)
//   - proofs the inclusion proofs for each node in accumulatorfrom in MMR(sizeB)
func ConsistentRoots(hasher hash.Hash, ifrom uint64, accumulatorfrom [][]byte, proofs [][][]byte) ([][]byte, error) {
	frompeaks := Peaks(ifrom)

	if len(frompeaks) != len(proofs) {
		return nil, ErrAccumulatorProofLen
	}

	roots := [][]byte{}

	for iacc := 0; iacc < len(accumulatorfrom); iacc++ {
		// remembering that peaks are 1 based (for now)
		root := IncludedRoot(hasher, frompeaks[iacc], accumulatorfrom[iacc], proofs[iacc])
		// The nature of MMR's is that many nodes are committed by the
		// same accumulator peak, and that peak changes with
		// low frequency.
		if len(roots) > 0 && bytes.Equal(roots[len(roots)-1], root) {
			continue
		}
		roots = append(roots, root)
	}

	return roots, nil
}
