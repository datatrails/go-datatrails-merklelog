package mmr

import (
	"bytes"
	"errors"
	"hash"
)

var (
	ErrAccumulatorProofLen = errors.New("a proof for each accumulator is required")
)

func ConsistentRoots(hasher hash.Hash, fromSize uint64, accumulatorfrom [][]byte, proofs [][][]byte) ([][]byte, error) {
	frompeaks := Peaks(fromSize)

	if len(frompeaks) != len(proofs) {
		return nil, ErrAccumulatorProofLen
	}

	roots := [][]byte{}

	for iacc := 0; iacc < len(accumulatorfrom); iacc++ {
		// remembering that peaks are 1 based (for now)
		root := IncludedRoot(hasher, frompeaks[iacc] - 1, accumulatorfrom[iacc], proofs[iacc])
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
