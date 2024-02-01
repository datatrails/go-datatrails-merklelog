package mmr

import (
	"crypto/sha256"
	"hash"
	"reflect"
	"testing"
)

type testStoreProver interface {
	indexStoreGetter
	Next() uint64
}

func TestGetRoot(t *testing.T) {

	hasher := sha256.New()
	db := NewCanonicalTestDB(t)

	// H return the node hash for index i from the canonical test tree.
	//
	// The canonical test tree has the hashes for all the positions, including
	// the interior nodes. Created by mandraulicaly hashing nodes so that tree
	// construction can legitimately be tested against it.
	H := func(i uint64) []byte {
		return db.mustGet(i)
	}
	Hrl := func(right, left []byte) []byte {
		hasher.Reset()
		hasher.Write(right)
		hasher.Write(left)
		return hasher.Sum(nil)
	}

	type args struct {
		mmrSize uint64
		store   indexStoreGetter
		hasher  hash.Hash
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		//	3            15
		//	           /    \
		//	          /      \
		//	         /        \
		//	2       7          14             22
		//	      /   \       /   \          /   \
		//	1    3     6    10     13      18      21      25
		//	    / \  /  \   / \   /  \    /  \    /  \    /   \
		//	0  1   2 4   5 8   9 11   12 16   17 19   20 23   24 26
		{
			"4 peaks (size=26)",
			args{
				26, db, hasher,
			},
			Hrl(Hrl(Hrl(H(26-1), H(25-1)), H(22-1)), H(15-1)),
			false,
		},
		{
			"3 peaks (size=25)",
			args{
				25, db, hasher,
			},
			Hrl(Hrl(H(25-1), H(22-1)), H(15-1)),
			false,
		},
		{
			"1 peaks (size=15)",
			args{
				15, db, hasher,
			},
			H(15 - 1),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetRoot(tt.args.mmrSize, tt.args.store, tt.args.hasher)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRoot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRoot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndexProofLocal(t *testing.T) {
	db := NewCanonicalTestDB(t)

	// H return the node hash for index i from the canonical test tree.
	//
	// The canonical test tree has the hashes for all the positions, including
	// the interior nodes. Created by mandraulically hasing nodes so that tree
	// concstruction can legitemately be tested against it.
	H := func(i uint64) []byte {
		return db.mustGet(i)
	}

	// the proof nodes for leaf 0
	h1 := H(1)
	h5 := H(5)
	h13 := H(13)
	// the additional proof nodes for leaf 1
	h0 := H(0)

	type args struct {
		store   testStoreProver
		i       uint64
		mmrSize uint64 // set zero to take from db
	}
	tests := []struct {
		name    string
		args    args
		want    [][]byte
		want1   uint64
		wantErr bool
	}{
		// the 0 based tree
		// 3              14
		//              /    \
		//             /      \
		//            /        \
		//           /          \
		// 2        6            13
		//        /   \        /    \
		// 1     2     5      9     12     17
		//      / \   / \    / \   /  \   /  \
		// 0   0   1 3   4  7   8 10  11 15  16 18

		{"2 (interior node)", args{
			db, 2, 26,
		}, [][]byte{H(5), H(13)}, 14, false},

		{"2 (interior node) smaller mmr", args{
			db, 2, 11,
		}, [][]byte{H(5)}, 6, false},

		{"0", args{
			db, 0, 26,
		}, [][]byte{h1, h5, h13}, 14, false},

		{"1", args{
			db, 1, 26,
		}, [][]byte{h0, h5, h13}, 14, false},

		{"3", args{
			db, 3, 26,
		}, [][]byte{H(4), H(2), H(13)}, 14, false},

		{"4", args{
			db, 4, 26,
		}, [][]byte{H(3), H(2), H(13)}, 14, false},

		{"7", args{
			db, 7, 26,
		}, [][]byte{H(8), H(12), H(6)}, 14, false},

		{"8", args{
			db, 8, 26,
		}, [][]byte{H(7), H(12), H(6)}, 14, false},

		{"10", args{
			db, 10, 26,
		}, [][]byte{H(11), H(9), H(6)}, 14, false},

		{"11", args{
			db, 11, 26,
		}, [][]byte{H(10), H(9), H(6)}, 14, false},

		// Notice: this is the isolated peak, hence the short length and the alternate root
		{"15", args{
			db, 15, 18,
		}, [][]byte{H(16)}, 17, false},

		{"16", args{
			db, 16, 18,
		}, [][]byte{H(15)}, 17, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mmrSize := tt.args.mmrSize
			if mmrSize == 0 {
				mmrSize = tt.args.store.Next()
			}
			got, got1, err := IndexProofLocal(mmrSize, tt.args.store, tt.args.i)
			if (err != nil) != tt.wantErr {
				t.Errorf("IndexProofLocal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IndexProofLocal() = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("IndexProofLocal() = %v, want %v", got1, tt.want1)
			}
		})
	}
}

// TestBagPeaksRHS
func TestBagPeaksRHS(t *testing.T) {

	db := NewCanonicalTestDB(t)

	// H return the node hash for index i from the canonical test tree.
	//
	// The canonical test tree has the hashes for all the positions, including
	// the interior nodes. Created by mandraulically hasing nodes so that tree
	// concstruction can legitemately be tested against it.
	H := func(i uint64) []byte {
		return db.mustGet(i)
	}

	type args struct {
		store  indexStoreGetter
		hasher hash.Hash
		pos    uint64
		peaks  []uint64
	}

	hasher := sha256.New()

	Hrl := func(right, left []byte) []byte {
		hasher.Reset()
		hasher.Write(right)
		hasher.Write(left)
		return hasher.Sum(nil)
	}

	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		//	3            15
		//	           /    \
		//	          /      \
		//	         /        \
		//	2       7          14             22
		//	      /   \       /   \          /   \
		//	1    3     6    10     13      18      21      25
		//	    / \  /  \   / \   /  \    /  \    /  \    /   \
		//	0  1   2 4   5 8   9 11   12 16   17 19   20 23   24 26

		{
			"all peaks to the right of the largest",
			args{
				db, hasher, 15, []uint64{22, 25, 26},
			}, Hrl(Hrl(H(26-1), H(25-1)), H(22-1)), false,
		},
		{
			"no peaks to the right, so nil return",
			args{
				db, hasher, 26, []uint64{22, 25, 26},
			}, nil, false,
		},
		{
			"one peak to the right, so return its exact value",
			args{
				db, hasher, 25, []uint64{22, 25, 26},
			}, H(26 - 1), false,
		},

		{
			"exactly two peaks to the right",
			args{
				db, hasher, 22, []uint64{22, 25, 26},
			}, Hrl(H(26-1), H(25-1)), false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BagPeaksRHS(tt.args.store, tt.args.hasher, tt.args.pos, tt.args.peaks)
			if (err != nil) != tt.wantErr {
				t.Errorf("BagPeaksRHS() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BagPeaksRHS() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLeftPosForHeight(t *testing.T) {
	type args struct {
		height uint64
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{"zero", args{0}, 0},
		{"one", args{1}, 2},
		{"two", args{2}, 6},
		{"three", args{3}, 14},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LeftPosForHeight(tt.args.height); got != tt.want {
				t.Errorf("LeftPosForHeight() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeightPeakRight(t *testing.T) {

	type args struct {
		mmrSize uint64
		height  uint64
		i       uint64
	}
	tests := []struct {
		name  string
		args  args
		want  uint64
		want1 uint64
		want2 bool
	}{
		// 3              14
		//              /    \
		//             /      \
		//            /        \
		//           /          \
		// 2        6            13
		//        /   \        /    \
		// 1     2     5      9     12     17     20
		//      / \   / \    / \   /  \   /  \
		// 0   0   1 3   4  7   8 10  11 15  16 18 | 19
		{"happy path 1", args{19, 1, 2}, 1, 5, true},
		// behaviour on the 'sad' path is undefined
		// {"sad path 2", args{19, 1, 5}, 1, 9, false}, // no right sibling
		{"happy path 3", args{19, 1, 9}, 1, 12, true},
		// {"sad path 4", args{19, 1, 12}, 1, 17, false}, no right sibling
		{"happy path 14", args{19, 3, 14}, 1, 17, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2 := HeightPeakRight(tt.args.mmrSize, tt.args.height, tt.args.i)
			if got2 != tt.want2 {
				t.Errorf("HeightPeakRight() got2 = %v, want %v", got2, tt.want2)
			}
			if got != tt.want {
				t.Errorf("HeightPeakRight() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("HeightPeakRight() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestPeaksLHS(t *testing.T) {

	db := NewCanonicalTestDB(t)

	// H gets a hash from store by *index*
	H := func(i uint64) []byte {
		return db.mustGet(i)
	}

	type args struct {
		store indexStoreGetter
		i     uint64
		peaks []uint64
	}
	tests := []struct {
		name    string
		args    args
		want    [][]byte
		wantErr bool
	}{
		// We get 'peaks' as one based positions, but we get the 'right' peak as
		// a zero based index. Lets visualise the tree as positions:
		//
		//	3            15
		//	           /    \
		//	          /      \
		//	         /        \
		//	2       7          14             22
		//	      /   \       /   \          /   \
		//	1    3     6    10     13      18      21      25
		//	    / \  /  \   / \   /  \    /  \    /  \    /   \
		//	0  1   2 4   5 8   9 11   12 16   17 19   20 23   24 26

		{
			"two left, one right",
			args{db, 24 /*index not pos*/, []uint64{15, 22, 25, 26}}, [][]byte{H(15 - 1), H(22 - 1)}, false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PeaksLHS(tt.args.store, tt.args.i, tt.args.peaks)
			if (err != nil) != tt.wantErr {
				t.Errorf("PeaksLHS() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PeaksLHS() = %v, want %v", got, tt.want)
			}
		})
	}
}
