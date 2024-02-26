package mmr

import (
	"crypto/sha256"
	"hash"
	"reflect"
	"testing"
)

func testMinimal(t *testing.T, hasher hash.Hash, store *testDb) {
	hasher.Reset()

	rootA, err := GetRoot(11, store, hasher)
	if err != nil {
		t.Errorf(": %v", err)
	}

	peakProof, err := IndexProof(11, store, hasher, 0)
	if err != nil {
		t.Errorf(": %v", err)
	}

	// mmrSize uint64, hasher hash.Hash, leafHash []byte, iLeaf uint64, proof [][]byte, root []byte,
	peakHash, err := store.Get(0)
	if err != nil {
		t.Errorf(": %v", err)
	}

	ok := VerifyInclusion(11, hasher, peakHash, 0, peakProof, rootA)
	if !ok {
		t.Errorf("it is not ok")
	}

	peakProof, err = IndexProof(11, store, hasher, 1)
	if err != nil {
		t.Errorf(": %v", err)
	}

	// mmrSize uint64, hasher hash.Hash, leafHash []byte, iLeaf uint64, proof [][]byte, root []byte,
	peakHash, err = store.Get(1)
	if err != nil {
		t.Errorf(": %v", err)
	}

	ok = VerifyInclusion(11, hasher, peakHash, 1, peakProof, rootA)
	if !ok {
		t.Errorf("it is not ok")
	}

	peakProof, err = IndexProof(11, store, hasher, 2)
	if err != nil {
		t.Errorf(": %v", err)
	}

	// mmrSize uint64, hasher hash.Hash, leafHash []byte, iLeaf uint64, proof [][]byte, root []byte,
	peakHash, err = store.Get(2)
	if err != nil {
		t.Errorf(": %v", err)
	}

	ok = VerifyInclusion(11, hasher, peakHash, 2, peakProof, rootA)
	if !ok {
		t.Errorf("it is not ok")
	}

	peakProof, err = IndexProof(11, store, hasher, 6)
	if err != nil {
		t.Errorf(": %v", err)
	}

	// mmrSize uint64, hasher hash.Hash, leafHash []byte, iLeaf uint64, proof [][]byte, root []byte,
	peakHash, err = store.Get(6)
	if err != nil {
		t.Errorf(": %v", err)
	}

	ok = VerifyInclusion(11, hasher, peakHash, 6, peakProof, rootA)
	if !ok {
		t.Errorf("it is not ok")
	}
}

func TestIndexConsistencyProof(t *testing.T) {

	hasher := sha256.New()
	store := NewGeneratedTestDB(t, 63)

	testMinimal(t, hasher, store)

	type args struct {
		mmrSizeA uint64
		mmrSizeB uint64
	}
	tests := []struct {
		name         string
		args         args
		wantProof    ConsistencyProof
		wantProofErr bool
		wantVerify   bool
	}{
		// TODO: Add test cases.
		{
			name: "11 to 18",
			args: args{
				mmrSizeA: 11,
				mmrSizeB: 18,
			},
			wantProof:    ConsistencyProof{},
			wantProofErr: false,
			wantVerify:   true,
		},
		{
			name: "7 to 15",
			args: args{
				mmrSizeA: 7,
				mmrSizeB: 15,
			},
			wantProof:    ConsistencyProof{},
			wantProofErr: false,
			wantVerify:   true,
		},
		{
			name: "7 to 63",
			args: args{
				mmrSizeA: 7,
				mmrSizeB: 63,
			},
			wantProof:    ConsistencyProof{},
			wantProofErr: false,
			wantVerify:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IndexConsistencyProof(tt.args.mmrSizeA, tt.args.mmrSizeB, store, hasher)
			if (err != nil) != tt.wantProofErr {
				t.Errorf("IndexConsistencyProof() error = %v, wantErr %v", err, tt.wantProofErr)
				return
			}
			if tt.wantProof.MMRSizeA != 0 && tt.wantProof.MMRSizeA != got.MMRSizeA {
				t.Errorf(
					"IndexConsistencyProof(), want MMRSizeA %d, got %d",
					tt.wantProof.MMRSizeA, got.MMRSizeA)
			}
			if tt.wantProof.MMRSizeB != 0 && tt.wantProof.MMRSizeB != got.MMRSizeB {
				t.Errorf(
					"IndexConsistencyProof(), want MMRSizeB %d, got %d",
					tt.wantProof.MMRSizeB, got.MMRSizeB)
			}
			if tt.wantProof.Path != nil && !reflect.DeepEqual(got.Path, tt.wantProof.Path) {
				t.Errorf("IndexConsistencyProof(), want %v, got %v", tt.wantProof.Path, got.Path)
			}

			// If the passing test doesn't produce a valid proof then we are done.
			if tt.wantProofErr == true {
				return
			}

			iPeaks := Peaks(got.MMRSizeA)
			peakHashesA, err := PeakBagRHS(store, hasher, 0, iPeaks)
			if err != nil {
				t.Errorf("PeakBagRHS: %v", err)
			}

			// Ordinarily, rootA would be from a previously signed merkle root
			// and only rootB for the current (proposed) log extension.
			rootA, err := GetRoot(got.MMRSizeA, store, hasher)
			if err != nil {
				t.Errorf("GetRoot: %v", err)
			}

			rootB, err := GetRoot(got.MMRSizeB, store, hasher)
			if err != nil {
				t.Errorf("GetRoot: %v", err)
			}

			verified := VerifyConsistency(hasher, peakHashesA, got, rootA, rootB)

			if tt.wantVerify != verified {
				t.Errorf("VerifyConsistency() = %v, expected %v", tt.wantVerify, verified)
			}
		})
	}

	// // H return the node hash for index i from the canonical test tree.
	// //
	// // The canonical test tree has the hashes for all the positions, including
	// // the interior nodes. Created by mandraulicaly hashing nodes so that tree
	// // construction can legitimately be tested against it.
	// H := func(i uint64) []byte {
	// 	return db.mustGet(i)
	// }
	// Hrl := func(right, left []byte) []byte {
	// 	hasher.Reset()
	// 	hasher.Write(right)
	// 	hasher.Write(left)
	// 	return hasher.Sum(nil)
	// }

}
