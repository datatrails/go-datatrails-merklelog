package mmr

import (
	"errors"
	"fmt"
)

var (
	ErrNewLogSizeMustBeGreater = errors.New("the new log size must be greater than the previous")
)

type ConsistencyProofLocal struct {
	LogIndex   uint64
	Path       [][]byte
	SizeA      uint64
	PeakIndexA uint64
	// The Path proof is identical in mmr size = A and size = B up to Path[HeightA]
	HeightA    uint64
	SizeB      uint64
	PeakIndexB uint64
}

// InclusionProofLocalExtend produces a proof which can verify for two mmr sizes
// It shows that the proof for mmrSizeB is an *extention* of the proof for
// mmrSizeA.
func InclusionProofLocalExtend(mmrSizeA, mmrSizeB uint64, store indexStoreGetter, i uint64) (ConsistencyProofLocal, error) {

	height := uint64(0)

	var err error
	var value []byte

	var prevComplete bool

	conLocal := ConsistencyProofLocal{
		LogIndex: i,
		SizeA:    mmrSizeA,
		SizeB:    mmrSizeB,
	}
	var proofPath [][]byte

	if mmrSizeB <= mmrSizeA {
		return ConsistencyProofLocal{}, fmt.Errorf(
			"%d is less than or equal %d: %w", mmrSizeB, mmrSizeA, ErrNewLogSizeMustBeGreater)
	}

	for i < mmrSizeB {
		iHeight := IndexHeight(i)
		iNextHeight := IndexHeight(i + 1)
		if iNextHeight > iHeight {
			iSibling := i - SiblingOffset(height)
			if iSibling >= mmrSizeA && !prevComplete {
				conLocal.PeakIndexA = i
				conLocal.HeightA = uint64(len(proofPath))
				prevComplete = true
			}
			if iSibling >= mmrSizeB {
				break
			}

			if value, err = store.Get(iSibling); err != nil {
				return ConsistencyProofLocal{}, err
			}
			proofPath = append(proofPath, value)
			// go to parent node
			i += 1
		} else {
			iSibling := i + SiblingOffset(height)
			if iSibling >= mmrSizeA && !prevComplete {
				conLocal.PeakIndexA = i
				conLocal.HeightA = uint64(len(proofPath))
				prevComplete = true
			}
			if iSibling >= mmrSizeB {
				break
			}

			if value, err = store.Get(iSibling); err != nil {
				return ConsistencyProofLocal{}, err
			}
			proofPath = append(proofPath, value)
			// goto parent node
			i += 2 << height
		}
		height += 1
	}
	conLocal.PeakIndexB = i
	conLocal.Path = proofPath
	return conLocal, nil
}
