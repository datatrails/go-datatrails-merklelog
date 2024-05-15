package mmr

import (
	"fmt"
	"math"
	"testing"
)

func TestJumpLeftPerfect(t *testing.T) {
	type args struct {
		pos uint64
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		//  3            15
		//             /    \
		//            /      \
		//           /        \
		//  2       7          14
		//        /   \       /   \
		//  1    3     6    10     13      18
		//      / \  /  \   / \   /  \    /  \
		//  0  1   2 4   5 8   9 11   12 16   17

		// this is the case used in the example at
		// https://github.com/mimblewimble/grin/blob/0ff6763ee64e5a14e70ddd4642b99789a1648a32/core/src/core/pmmr.rs#L606
		{"13", args{13}, 6},
		// note this is based on the same perfect tree as 13, and jumps to the
		// equivelent node in the perfect tree to the left, which is 3
		{"10", args{10}, 3},
		{"6", args{6}, 3},
		// Notice that the perfect tree containing 18 is infact a sibling tree
		// to the perfect tree rooted at 15. And so 18's partner node on this
		// level is 3 directly
		{"18", args{18}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JumpLeftPerfect(tt.args.pos); got != tt.want {
				t.Errorf("JumpLeftRightMost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJumpRightSibling(t *testing.T) {
	type args struct {
		i uint64
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		//  3            15
		//             /    \
		//            /      \
		//           /        \
		//  2       7          14              22
		//        /   \       /   \
		//  1    3     6    10     13      18     21
		//      / \  /  \   / \   /  \    /  \
		//  0  1   2 4   5 8   9 11   12 16   17 19 20
		{"10", args{10}, 13},
		// note that 6 (above) doesn't actually have a 'right' sibling so the
		// method returns a meaningless result
		{"6", args{6}, 9},
		{"1", args{1}, 2},
		// 2 has no right sibling
		{"4", args{4}, 5},
		// 5 has no right sibling
		{"8", args{8}, 9},
		// 9 has no right sibling
		{"11", args{11}, 12},
		// 12 has no right sibling
		{"16", args{16}, 17},
		{"3", args{3}, 6},
		// 6 has no right sibling
		{"10", args{10}, 13},
		// 13 has no right sibling
		{"18", args{18}, 21},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JumpRightSibling(tt.args.i); got != tt.want {
				t.Errorf("JumpRight() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndexHeight(t *testing.T) {
	type args struct {
		pos uint64
	}
	tests := []struct {
		name string
		args args
		want uint64
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

		{
			"nine is 1", args{9}, 1,
		},
		{
			"eleven is 0", args{11}, 0,
		},
		{
			"fifteen is 0", args{11}, 0,
		},
		{
			"twelve is 1", args{12}, 1,
		},
		{
			"thirteen is 2", args{13}, 2,
		},
		{
			"twenty one is 2", args{21}, 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IndexHeight(tt.args.pos); got != tt.want {
				t.Errorf("PosHeight() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPosHeight(t *testing.T) {
	type args struct {
		pos uint64
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		//  3            15
		//             /    \
		//            /      \
		//           /        \
		//  2       7          14
		//        /   \       /   \
		//  1    3     6    10     13      18
		//      / \  /  \   / \   /  \    /  \
		//  0  1   2 4   5 8   9 11   12 16   17
		{"ten is 1", args{10}, 1},
		{"twelve is 0", args{12}, 0},
		{"thirteen is 1", args{13}, 1},
		{"fourteen is 2", args{14}, 2},
		{"twenty two is 2", args{22}, 2},
		{"fifteen is 3", args{15}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PosHeight(tt.args.pos); got != tt.want {
				t.Errorf("PosHeight() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLeftChild(t *testing.T) {
	type args struct {
		i uint64
	}
	tests := []struct {
		name  string
		args  args
		want  uint64
		want1 bool
	}{
		//  3            15
		//             /    \
		//            /      \
		//           /        \
		//  2       7          14
		//        /   \       /   \
		//  1    3     6    10     13      18
		//      / \  /  \   / \   /  \    /  \
		//  0  1   2 4   5 8   9 11   12 16   17
		{"3", args{3}, 1, true},
		{"7", args{7}, 3, true},
		{"6", args{6}, 4, true},
		{"14", args{14}, 10, true},
		{"8", args{8}, 0, false},
		{"1", args{1}, 0, false},
		{"2", args{2}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := LeftChild(tt.args.i)
			if got != tt.want {
				t.Errorf("LeftChild() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("LeftChild() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestBitLength(t *testing.T) {
	type args struct {
		num uint64
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{"thirteen is four", args{num: 13}, 4},
		{"max uint64 is 64", args{num: math.MaxUint64}, 64},
		{"one", args{num: 1}, 1},
		{"two", args{num: 2}, 2},
		{"three is two", args{num: 3}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BitLength64(tt.args.num); got != tt.want {
				t.Errorf("BitLength() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaxPeakHeight(t *testing.T) {
	type args struct {
		i uint64
	}
	tests := []struct {
		args args
		want uint64
	}{
		{args{0}, 0},
		{args{9}, 2},
		{args{13}, 2},
		{args{14}, 3},
		{args{17}, 3},
		{args{27}, 3},
		{args{30}, 4},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("mmr index %d = max peak height %d", tt.args.i, tt.want), func(t *testing.T) {
			if got := MaxPeakHeight(tt.args.i); got != tt.want {
				t.Errorf("MaxPeakHeight() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeightIndexLeafCount(t *testing.T) {
	type args struct {
		heightIndex uint64
	}
	tests := []struct {
		args args
		want uint64
	}{
		{args{0}, 1},
		{args{1}, 2},
		{args{2}, 4},
		{args{3}, 8},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("mmr index %d = leaf count %d", tt.args.heightIndex, tt.want), func(t *testing.T) {
			if got := HeightIndexLeafCount(tt.args.heightIndex); got != tt.want {
				t.Errorf("HeightIndexLeafCount() = %v, want %v", got, tt.want)
			}
		})
	}
}
