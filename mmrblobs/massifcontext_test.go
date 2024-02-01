package mmrblobs

import (
	"testing"
)

func TestMassifMaxCount(t *testing.T) {
	type args struct {
		height uint8
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{"height 3", args{3}, (8 - 1) * LogEntryBytes},
		{"height 4", args{4}, (16 - 1) * LogEntryBytes},
		{"height 8", args{8}, (256 - 1) * LogEntryBytes},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TreeSize(tt.args.height); got != tt.want {
				t.Errorf("TreeSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMassifLastLeafIndex(t *testing.T) {
	type args struct {
		firstIndex uint64
		height     uint8
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{"m0, height 2", args{0, 2}, 1},
		{"m1, height 2", args{3, 2}, 4},
		{"m2, height 2", args{7, 2}, 8},

		{"m0, height 3", args{0, 3}, 4},
		{"m1, height 3", args{7, 3}, 11},
		{"m2, height 3", args{15, 3}, 19},

		{"m0, height 4", args{0, 4}, 11},
		{"m1, height 4", args{15, 4}, 26},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RangeLastLeafIndex(tt.args.firstIndex, tt.args.height); got != tt.want {
				t.Errorf("MassifLastLeafIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMassifRootIndex(t *testing.T) {
	type args struct {
		firstIndex uint64
		height     uint8
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{"m0, height 2", args{0, 2}, 3 - 1},
		{"m1, height 2", args{3, 2}, 6 - 1},
		{"m2, height 2", args{7, 2}, 10 - 1},

		{"m0, height 3", args{0, 3}, 7 - 1},
		{"m1, height 3", args{7, 3}, 14 - 1},
		{"m2, height 3", args{15, 3}, 22 - 1},

		{"m0, height 4", args{0, 4}, 15 - 1},
		{"m1, height 4", args{15, 4}, 30 - 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RangeRootIndex(tt.args.firstIndex, tt.args.height); got != tt.want {
				t.Errorf("MassifRootIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}
