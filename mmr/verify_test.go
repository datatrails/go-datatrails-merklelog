package mmr

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVerifyLeavesIn38Bagged check that we can obtain and verify proofs for all 38 leaves
func TestVerifyLeavesIn38(t *testing.T) {
	hasher := sha256.New()
	db := NewCanonicalTestDB(t)
	mmrMaxSize := db.Next()
	numLeafs := LeafCount(mmrMaxSize)

	for iLeaf := uint64(0); iLeaf < numLeafs; iLeaf++ {

		mmrIndex := MMRIndex(iLeaf)

		for s := FirstMMRSize(MMRIndex(iLeaf)); s <= mmrMaxSize; s = FirstMMRSize(s + 1) {

			// Verify each leaf in all complete mmr sizes up to the size of the canonical mmr
			// for iLeaf := uint64(0); iLeaf < numLeafs; iLeaf++ {

			proof, err := InclusionProof(db, s-1, mmrIndex)
			require.NoError(t, err)

			nodeHash, err := db.Get(mmrIndex)
			require.NoError(t, err)

			accumulator, err := PeakHashes(db, s-1)
			require.NoError(t, err)
			iacc := PeakIndex(LeafCount(s), len(proof))
			require.Less(t, iacc, len(accumulator))

			peak := accumulator[iacc]
			root := IncludedRoot(hasher, mmrIndex, nodeHash, proof)
			if !bytes.Equal(root, peak) {
				fmt.Printf("%d %d TestVerifyLeavesIn38 failed\n", mmrIndex, iLeaf)
			}
			assert.Equal(t, root, peak)
		}
	}
	// fmt.Printf("VerifyInclusion() ok size=%d, leaves=%d, ok=%d\n", mmrSize, numLeafs, verifiedOk)
}
