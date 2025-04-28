package mmr

import (
	"testing"
)

func TestLog2Uint64(t *testing.T) {
	type args struct {
		num uint64
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{"1 -> 0", args{1}, 0},
		{"2 -> 1", args{2}, 1},
		{"3 -> 1", args{3}, 1},
		{"4 -> 2", args{4}, 2},
		{"8 -> 3", args{8}, 3},
		{"16 -> 4", args{16}, 4},
		{"17 -> 4", args{17}, 4},
		{"18 -> 4", args{18}, 4},
		{"19 -> 4", args{19}, 4},
		{"32 -> 5", args{32}, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Log2Uint64(tt.args.num); got != tt.want {
				t.Errorf("Log2Uint64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLog2Uint32(t *testing.T) {
	type args struct {
		num uint32
	}
	tests := []struct {
		name string
		args args
		want uint32
	}{
		{"1 -> 0", args{1}, 0},
		{"2 -> 1", args{2}, 1},
		{"3 -> 1", args{3}, 1},
		{"4 -> 2", args{4}, 2},
		{"8 -> 3", args{8}, 3},
		{"16 -> 4", args{16}, 4},
		{"17 -> 4", args{17}, 4},
		{"18 -> 4", args{18}, 4},
		{"19 -> 4", args{19}, 4},
		{"32 -> 5", args{32}, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Log2Uint32(tt.args.num); got != tt.want {
				t.Errorf("Log2Uint64() = %v, want %v", got, tt.want)
			}
		})
	}
}
