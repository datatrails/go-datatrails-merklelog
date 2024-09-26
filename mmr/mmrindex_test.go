package mmr

import (
	"testing"
)

func TestMMRIndex(t *testing.T) {
	tests := []struct {
		leafIndex uint64
		expected  uint64
	}{
		{0, 0},
		{1, 1},
		{2, 3},
		{3, 4},
		{4, 7},
		{5, 8},
		{6, 10},
		{7, 11},
		{8, 15},
		{9, 16},
		{10, 18},
		{11, 19},
		{12, 22},
		{13, 23},
		{14, 25},
		{15, 26},
		{16, 31},
		{17, 32},
		{18, 34},
		{19, 35},
		{20, 38},
	}

	for _, test := range tests {
		result := MMRIndex(test.leafIndex)
		if result != test.expected {
			t.Errorf("MMRIndex(%d) = %d; expected %d", test.leafIndex, result, test.expected)
		}
	}
}
