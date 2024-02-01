package mmr

import (
	"crypto/sha256"
	"hash"
	"testing"
)

func TestAddHashedLeaf(t *testing.T) {
	type args struct {
		store        NodeAppender
		hasher       hash.Hash
		hashedLeaves [][]byte
	}
	tests := []struct {
		name    string
		args    args
		want    uint64
		wantErr bool
	}{
		{
			"add four to empty db, adding the 2nd adds one new peak, adding the 4rth creates an additional 2 peaks",
			args{
				NewTestDb(t),
				sha256.New(),
				[][]byte{{1}, {2}, {3}, {4}},
			},
			7,
			false,
		},

		{
			"add three to empty db, note that the third does not create a new peak",
			args{
				NewTestDb(t),
				sha256.New(),
				[][]byte{{1}, {2}, {3}},
			},
			4,
			false,
		},
		{
			"add two to empty db, creates new peak at index 2",
			args{
				NewTestDb(t),
				sha256.New(),
				[][]byte{{1}, {2}},
			},
			3,
			false,
		},
		{
			"add one to empty db, no peaks, edge case",
			args{
				NewTestDb(t),
				sha256.New(),
				[][]byte{{1}},
			},
			1,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			var got uint64
			for _, value := range tt.args.hashedLeaves {
				if got, err = AddHashedLeaf(
					tt.args.store, tt.args.hasher, value); (err != nil) != tt.wantErr {
					t.Errorf("AddHashedLeaf() = %v, want %v, err: %v, wantErr: %v", got, tt.want, err, (err != nil))
				}
			}
			if got != tt.want {
				t.Errorf("AddHashedLeaf() = %v, want %v", got, tt.want)
			}
		})
	}
}
