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

// TestVerifyLeavesIn38Bagged check that we can obtain and verify proofs for all 38 leaves
func TestVerifyLeavesIn38Bagged(t *testing.T) {
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
		iNode := MMRIndex(iLeaf)

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

// TestVerify38Bagged check that we can obtain and verify proofs for all 38 *nodes*
func TestVerify38Bagged(t *testing.T) {
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
		// iNode := MMRIndex(iLeaf)

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

// TestVerifyPerfectRootsBagged checks we can produce and verify proofs for the
// perfect peaks, which should be just the peaks them selves
func TestVerifyPerfectRootsBagged(t *testing.T) {
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

func TestVerifyIndex30InSize63Bagged(t *testing.T) {

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
func TestReVerify38ForAllSizesBagged(t *testing.T) {
	hasher := sha256.New()
	// db := NewCanonicalTestDB(t)
	db := NewGeneratedTestDB(t, 63)
	maxMMRSize := db.Next()
	numLeafs := LeafCount(maxMMRSize)

	for iLeaf := uint64(0); iLeaf < numLeafs; iLeaf++ {

		iNode := MMRIndex(iLeaf)

		// Check that all valid mmr sizes which contain the node can generate verifiable proofs for it.
		//
		// iLeaf is the leaf we are interested in ensuring verification for.
		// jLeaf is used to derive all the successive mmrSizes that continue to contain iLeaf
		for jLeaf := iLeaf; jLeaf < numLeafs; jLeaf++ {
			// the spur length + the node index gives us the minimum mmrsize that contains the leaf
			jNode := MMRIndex(jLeaf)
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

	getProofBagged := func(mmrSize uint64, i uint64) [][]byte {
		proof, err := IndexProofBagged(mmrSize, db, hasher, i)
		require.NoError(t, err)
		if mmrSize == 1 && proof != nil {
			t.Errorf("IndexProof() err: %v", errors.New("mmr size 1 should return nil proof"))
			return nil
		}
		return proof
	}
	getProof := func(mmrSize uint64, i uint64) [][]byte {
		proof, err := IndexProof(db, mmrSize-1, i)
		require.NoError(t, err)
		if mmrSize == 1 && proof != nil {
			t.Errorf("IndexProof() err: %v", errors.New("mmr size 1 should return nil proof"))
			return nil
		}
		return proof
	}

	verifyBagged := func(mmrSize uint64, nodeHash []byte, mmrIndex uint64, proof [][]byte) bool {
		root, err := GetRoot(mmrSize, db, hasher)
		require.NoError(t, err)
		if mmrSize == 1 {
			// special case
			return proof == nil
		}
		baggedOk := VerifyInclusionBagged(mmrSize, hasher, nodeHash, mmrIndex, proof, root)
		return baggedOk
		// ok, lenProofUsed := VerifyInclusionPath(mmrSize, hasher, nodeHash, iNode, proof, root)
		// return baggedOk && ok && lenProofUsed == len(proof)
	}
	verify := func(mmrSize uint64, nodeHash []byte, mmrIndex uint64, proof [][]byte) (bool, int) {

		// To account for interior nodes, we add the height of the node to the proof length.
		nodeHeightIndex := IndexHeight(mmrIndex)
		d := len(proof) + int(nodeHeightIndex)

		// get the index into the accumulator
		// peakMap is also the leaf count, which is often also known
		peakMap := LeafCount(mmrSize)
		peakIndex := PeakIndex(peakMap, d)
		peakHashes, err := PeakHashes(db, mmrSize)
		require.Less(t, peakIndex, len(peakHashes))
		require.NoError(t, err)
		root := peakHashes[peakIndex]

		return VerifyInclusionPath(mmrSize, hasher, nodeHash, mmrIndex, proof, root)
	}

	type proofNodes struct {
		iLocalPeak       uint64
		localHeightIndex uint64
		local            []uint64
		peaksRHS         []uint64
		peaksLHS         []uint64
	}

	type args struct {
		mmrSize     uint64
		leafHash    []byte
		mmrIndex    uint64
		proofBagged [][]byte
		proof       [][]byte
	}
	tests := []struct {
		name             string
		args             args
		want             bool
		expectProofNodes *proofNodes
	}{
		{
			"prove node index 0 in MMR(3)",
			args{3, H(0), 0, getProofBagged(3, 0), getProof(3, 0)}, true, nil,
		},
		{
			"prove node index 0 in MMR(7)",
			args{7, H(0), 0, getProofBagged(7, 0), getProof(7, 0)}, true, nil,
		},

		{
			"prove interior node index 2",
			args{26, H(2), 2, getProofBagged(26, 2), getProof(26, 2)}, true, nil,
		},

		{
			"prove leaf node index 23 for sz 25",
			args{25, H(23), 23, getProofBagged(25, 23), getProof(25, 23)},
			true,
			&proofNodes{
				iLocalPeak:       24,
				localHeightIndex: 1,
				local:            []uint64{22},
				peaksLHS:         []uint64{14, 21},
				peaksRHS:         nil,
			},
		},

		{
			"prove leaf node index 7 for sz 11",
			args{11, H(7), 7, getProofBagged(11, 7), getProof(11, 7)},
			true,
			&proofNodes{
				iLocalPeak:       9,
				localHeightIndex: 1,
				local:            []uint64{8},
				peaksLHS:         []uint64{6},
				peaksRHS:         []uint64{10},
			},
		},
		{
			"prove leaf node index 7 for sz 19",
			args{19, H(7), 7, getProofBagged(19, 7), getProof(19, 7)},
			true,
			&proofNodes{
				iLocalPeak:       14,
				localHeightIndex: 3,
				local:            []uint64{8, 12, 6},
				peaksLHS:         nil,
				peaksRHS:         []uint64{17, 18},
			},
		},

		{ // this fails
			"prove leaf node index 22 for sz 26",
			args{26, H(22), 22, getProofBagged(26, 22), getProof(26, 22)},
			true,
			&proofNodes{
				iLocalPeak:       24,
				localHeightIndex: 1,
				local:            []uint64{23},
				peaksLHS:         []uint64{14, 21},
				peaksRHS:         []uint64{25},
			},
		},

		{ // this is ok
			"prove leaf node index 19 for sz 26",
			args{26, H(19), 19, getProofBagged(26, 19), getProof(26, 19)}, true,
			&proofNodes{
				iLocalPeak:       21,
				localHeightIndex: 2,
				local:            []uint64{18, 17},
				peaksLHS:         []uint64{14},
				peaksRHS:         []uint64{24, 25},
			},
		},

		{
			"prove leaf node index 23 for sz 26",
			args{26, H(23), 23, getProofBagged(26, 23), getProof(26, 23)}, true, nil,
		},
		{
			"prove leaf node index 19 for sz 26",
			args{26, H(19), 19, getProofBagged(26, 19), getProof(26, 19)}, true, nil,
		},

		{
			"prove leaf node index 1",
			args{26, H(1), 1, getProofBagged(26, 1), getProof(26, 1)}, true, nil,
		},

		{
			"prove mid range (sibling mountains either side)",
			args{26, H(17 - 1), 16, getProofBagged(26, 16), getProof(26, 16)}, true, nil,
		},
		{
			"edge case, prove the solo leaf at the end of the range",
			args{39, H(26 - 1), 25, getProofBagged(39, 25), getProof(39, 25)}, true, nil,
		},
		{
			"edge case, prove the first leaf in the tree",
			args{26, H(0), 0, getProofBagged(26, 0), getProof(26, 0)}, true, nil,
		},
		{
			"edge case, prove a singleton",
			args{1, H(0), 1, getProofBagged(1, 0), getProof(1, 0)}, true, nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectProofNodes != nil {
				localPath, iLocalPeak, err := IndexProofLocal(
					tt.args.mmrSize, db, tt.args.mmrIndex)
				localHeightIndex := len(localPath)
				require.NoError(t, err)
				assert.Equal(t, tt.expectProofNodes.iLocalPeak, iLocalPeak, "local peak incorrect")
				assert.Equal(t, getNodes(db, tt.expectProofNodes.local...), localPath)

				peaks := PosPeaks(tt.args.mmrSize)

				peakBits := PeaksBitmap(tt.args.mmrSize)

				// the index into the packed accumulator peaks is the count how
				// many bits are set *above* localHeightIndex in the mask
				iPeak := PeakIndex(peakBits, len(localPath))
				// iPeak := bits.OnesCount64(peakBits & ^((1<<localHeightIndex)-1)) - 1
				fmt.Printf("%04b & %04b = %04b, m: %04b, iPeak: %d\n", peakBits, 1<<localHeightIndex, peakBits&(1<<localHeightIndex), (1<<localHeightIndex)-1, iPeak)
				assert.NotZero(t, peakBits&(1<<localHeightIndex), "peakBits doesn't contain local peak")

				peakHashes, err := PeakBagRHS(db, hasher, iLocalPeak+1, peaks)
				require.NoError(t, err)
				assert.Equal(t, getNodes(db, tt.expectProofNodes.peaksRHS...), peakHashes)

				leftPath, err := PeaksLHS(db, iLocalPeak+1, peaks)
				require.NoError(t, err)
				assert.Equal(t, getNodes(db, tt.expectProofNodes.peaksLHS...), leftPath)
			}
			if got := verifyBagged(tt.args.mmrSize, tt.args.leafHash, tt.args.mmrIndex, tt.args.proofBagged); got != tt.want {
				t.Errorf("Verify() = %v, want %v", got, tt.want)
			}

			if got, _ := verify(tt.args.mmrSize, tt.args.leafHash, tt.args.mmrIndex, tt.args.proof); got != tt.want {
				t.Errorf("Verify() = %v, want %v", got, tt.want)
			}

		})
	}
}
