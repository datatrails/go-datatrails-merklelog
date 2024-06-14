package mmr

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getNodes(db *testDb, iNodes ...uint64) [][]byte {
	var hashes [][]byte
	for i := 0; i < len(iNodes); i++ {
		hashes = append(hashes, db.mustGet(iNodes[i]))
	}
	return hashes
}

// TestVerifyLeavesIn38 check that we can obtain and verify proofs for all 38 leaves
func TestVerifyLeavesIn38(t *testing.T) {
	hasher := sha256.New()
	db := NewCanonicalTestDB(t)
	mmrSize := db.Next()
	numLeafs := LeafCount(mmrSize)

	root, err := GetRoot(mmrSize, db, hasher)
	if err != nil {
		t.Errorf("GetRoot() err: %v", err)
	}

	verifiedOk := uint64(0)
	for iLeaf := uint64(0); iLeaf < numLeafs; iLeaf++ {
		// for iLeaf := uint64(0); iLeaf < numLeafs; iLeaf++ {
		iNode := TreeIndex(iLeaf)

		proof, err := IndexProofBagged(mmrSize, db, hasher, iNode)
		require.NoError(t, err)

		nodeHash, err := db.Get(iNode)
		require.NoError(t, err)

		if !VerifyInclusionBagged(mmrSize, hasher, nodeHash, iNode, proof, root) {
			fmt.Printf("%d %d VerifyInclusion() failed\n", iNode, iLeaf)
		} else {
			verifiedOk++
		}
	}
	assert.Equal(t, verifiedOk, numLeafs)
	// fmt.Printf("VerifyInclusion() ok size=%d, leaves=%d, ok=%d\n", mmrSize, numLeafs, verifiedOk)
}

// TestVerify38 check that we can obtain and verify proofs for all 38 *nodes*
func TestVerify38(t *testing.T) {
	hasher := sha256.New()
	db := NewCanonicalTestDB(t)
	mmrSize := db.Next()

	root, err := GetRoot(mmrSize, db, hasher)
	if err != nil {
		t.Errorf("GetRoot() err: %v", err)
	}

	verifiedOk := uint64(0)
	for iNode := uint64(0); iNode < mmrSize; iNode++ {
		// for iLeaf := uint64(0); iLeaf < numLeafs; iLeaf++ {
		// iNode := TreeIndex(iLeaf)

		proof, err := IndexProofBagged(mmrSize, db, hasher, iNode)
		require.NoError(t, err)

		nodeHash, err := db.Get(iNode)
		require.NoError(t, err)

		if !VerifyInclusionBagged(mmrSize, hasher, nodeHash, iNode, proof, root) {
			fmt.Printf("%d %d VerifyInclusion() failed\n", iNode, iNode)
		} else {
			verifiedOk++
		}
	}
	assert.Equal(t, verifiedOk, mmrSize)
	// fmt.Printf("VerifyInclusion() ok size=%d, leaves=%d, ok=%d\n", mmrSize, numLeafs, verifiedOk)
}

// TestVerifyPerfectRoots checks we can produce and verify proofs for the
// perfect peaks, which should be just the peaks them selves
func TestVerifyPerfectRoots(t *testing.T) {
	hasher := sha256.New()

	verifiedOk := 0

	sizes := []uint64{3, 7, 15, 31, 63}
	for _, mmrSize := range sizes {
		db := NewGeneratedTestDB(t, mmrSize)

		root, err := GetRoot(mmrSize, db, hasher)
		if err != nil {
			t.Errorf("GetRoot() err: %v", err)
		}

		iNode := mmrSize - 1
		proof, err := IndexProofBagged(mmrSize, db, hasher, iNode)
		require.NoError(t, err)

		nodeHash, err := db.Get(iNode)
		require.NoError(t, err)

		if !VerifyInclusionBagged(mmrSize, hasher, nodeHash, iNode, proof, root) {
			fmt.Printf("%d %d VerifyInclusion() failed\n", iNode, iNode)
		} else {
			verifiedOk++
		}
	}
	assert.Equal(t, verifiedOk, len(sizes))
	// fmt.Printf("VerifyInclusion() ok size=%d, leaves=%d, ok=%d\n", mmrSize, numLeafs, verifiedOk)
}

func TestVerifyIndex30InSize63(t *testing.T) {

	hasher := sha256.New()
	// 63 is the first mmr with a hieght of 5 (and so is a perfect peak)
	db := NewGeneratedTestDB(t, 63)
	root, err := GetRoot(63, db, hasher)
	require.NoError(t, err)
	peakProof, err := IndexProofBagged(63, db, hasher, 30)
	require.NoError(t, err)
	peakHash := db.mustGet(30)
	ok := VerifyInclusionBagged(63, hasher, peakHash, 30, peakProof, root)
	assert.True(t, ok)
}

// TestReVerify38ForAllSizes
// Test that as the mmr grows, the previously verified nodes continue to be
// provable and verifiable.  Note that the proofs will be different as the tree
// root changes with the size. However, note also that any historic proof can be
// shown to be a 'sub-proof' of the new accumulator state and hence verifiable
// or exchangeable at any time.
// bug-9026
func TestReVerify38ForAllSizes(t *testing.T) {
	hasher := sha256.New()
	// db := NewCanonicalTestDB(t)
	db := NewGeneratedTestDB(t, 63)
	maxMMRSize := db.Next()
	numLeafs := LeafCount(maxMMRSize)

	for iLeaf := uint64(0); iLeaf < numLeafs; iLeaf++ {

		iNode := TreeIndex(iLeaf)

		// Check that all valid mmr sizes which contain the node can generate verifiable proofs for it.
		//
		// iLeaf is the leaf we are interested in ensuring verification for.
		// jLeaf is used to derive all the successive mmrSizes that continue to contain iLeaf
		for jLeaf := iLeaf; jLeaf < numLeafs; jLeaf++ {
			// the spur length + the node index gives us the minimum mmrsize that contains the leaf
			jNode := TreeIndex(jLeaf)
			spurLen := SpurHeightLeaf(jLeaf)

			jMMRSize := jNode + spurLen + 1

			root, err := GetRoot(jMMRSize, db, hasher)
			require.NoError(t, err)
			// Get the proof for *** iLeaf's node ***
			proof, err := IndexProofBagged(jMMRSize, db, hasher, iNode)
			require.NoError(t, err)
			if proof == nil {
				// This is the iLeaf == 0 && mmrSize == 1 case which is
				// peculiar. We can't really say the mmr with a single entry is
				// 'provable', it just is. In reality, a customer may create a
				// single event. They will get an empty receipt if they ask.
				// After the next confirmation tick, forestrie will sign a
				// tenant tree root. And in this case that root hash will just
				// be the single node. In this specific case, data trails
				// attestation is just the signed root. This peculiar case goes
				// away as soon as the second event is recorded.
				assert.Equal(t, db.mustGet(iNode), root)
				assert.Equal(t, iNode, uint64(0))
				assert.Equal(t, jMMRSize, uint64(1))
			} else {
				nodeHash, err := db.Get(iNode)
				require.NoError(t, err)

				// verify iNode using the j mmr size.
				ok := VerifyInclusionBagged(jMMRSize, hasher, nodeHash, iNode, proof, root)
				assert.Equal(t, ok, true)

			}
		}
	}
}

func TestVerify(t *testing.T) {

	hasher := sha256.New()
	db := NewCanonicalTestDB(t)
	// mmrSize := uint64(39)

	H := func(i uint64) []byte {
		return db.mustGet(i)
	}

	getProof := func(mmrSize uint64, i uint64) [][]byte {
		proof, err := IndexProofBagged(mmrSize, db, hasher, i)
		require.NoError(t, err)
		if mmrSize == 1 && proof != nil {
			t.Errorf("IndexProof() err: %v", errors.New("mmr size 1 should return nil proof"))
			return nil
		}
		return proof
	}

	verify := func(mmrSize uint64, nodeHash []byte, iNode uint64, proof [][]byte) bool {
		root, err := GetRoot(mmrSize, db, hasher)
		require.NoError(t, err)
		if mmrSize == 1 {
			// special case
			return proof == nil
		}
		baggedOk := VerifyInclusionBagged(mmrSize, hasher, nodeHash, iNode, proof, root)
		return baggedOk
		// ok, lenProofUsed := VerifyInclusionPath(mmrSize, hasher, nodeHash, iNode, proof, root)
		// return baggedOk && ok && lenProofUsed == len(proof)
	}

	type proofNodes struct {
		iLocalPeak uint64
		local      []uint64
		peaksRHS   []uint64
		peaksLHS   []uint64
	}

	type args struct {
		mmrSize  uint64
		leafHash []byte
		iLeaf    uint64
		proof    [][]byte
	}
	tests := []struct {
		name             string
		args             args
		want             bool
		expectProofNodes *proofNodes
	}{
		{ // this fails
			"prove leaf index 22 for sz 26",
			args{26, H(22), 22, getProof(26, 22)},
			true,
			&proofNodes{
				iLocalPeak: 24,
				local:      []uint64{23},
				peaksRHS:   []uint64{25},
				peaksLHS:   []uint64{14, 21},
			},
		},

		{ // this is ok
			"prove leaf index 19 for sz 26",
			args{26, H(19), 19, getProof(26, 19)}, true,
			&proofNodes{
				iLocalPeak: 21,
				local:      []uint64{18, 17},
				peaksRHS:   []uint64{24, 25},
				peaksLHS:   []uint64{14},
			},
		},
		{
			"prove leaf index 23 for sz 25",
			args{25, H(23), 23, getProof(25, 23)},
			true,
			&proofNodes{
				iLocalPeak: 24,
				local:      []uint64{22},
				peaksRHS:   nil,
				peaksLHS:   []uint64{14, 21},
			},
		},
		{
			"prove leaf index 23 for sz 26",
			args{26, H(23), 23, getProof(26, 23)}, true, nil,
		},
		{
			"prove leaf index 19 for sz 26",
			args{26, H(19), 19, getProof(26, 19)}, true, nil,
		},

		{
			"prove interior node index 2",
			args{26, H(2), 2, getProof(26, 2)}, true, nil,
		},
		{
			"prove leaf index 1",
			args{26, H(1), 1, getProof(26, 1)}, true, nil,
		},

		{
			"prove mid range (sibling mountains either side)",
			args{26, H(17 - 1), 16, getProof(26, 16)}, true, nil,
		},
		{
			"edge case, prove the solo leaf at the end of the range",
			args{39, H(26 - 1), 25, getProof(39, 25)}, true, nil,
		},
		{
			"edge case, prove the first leaf in the tree",
			args{26, H(0), 0, getProof(26, 0)}, true, nil,
		},
		{
			"edge case, prove a singleton",
			args{1, H(0), 1, getProof(1, 0)}, true, nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectProofNodes != nil {
				localPath, iLocalPeak, _, err := IndexProofPath(tt.args.mmrSize, db, tt.args.iLeaf)
				require.NoError(t, err)
				assert.Equal(t, iLocalPeak, tt.expectProofNodes.iLocalPeak, "local peak incorrect")
				assert.Equal(t, localPath, getNodes(db, tt.expectProofNodes.local...))

				peaks := Peaks(tt.args.mmrSize)

				peakHashes, err := PeakBagRHS(db, hasher, iLocalPeak+1, peaks)
				require.NoError(t, err)
				assert.Equal(t, peakHashes, getNodes(db, tt.expectProofNodes.peaksRHS...))

				leftPath, err := PeaksLHS(db, iLocalPeak+1, peaks)
				require.NoError(t, err)
				assert.Equal(t, leftPath, getNodes(db, tt.expectProofNodes.peaksLHS...))
			}
			if got := verify(tt.args.mmrSize, tt.args.leafHash, tt.args.iLeaf, tt.args.proof); got != tt.want {
				t.Errorf("Verify() = %v, want %v", got, tt.want)
			}
		})
	}
}
