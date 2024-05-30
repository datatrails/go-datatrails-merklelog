package mmr

import (
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
)

func TestLeafCount(t *testing.T) {
	type args struct {
		size uint64
	}
	tests := []struct {
		name   string
		args   args
		leaves uint64
	}{
		// 3              14
		//              /    \
		//             /      \
		//            /        \
		//           /          \
		// 2        6            13           21
		//        /   \        /    \
		// 1     2     5      9     12     17     20     24
		//      / \   / \    / \   /  \   /  \
		// 0   0   1 3   4  7   8 10  11 15  16 18  19 22  23   25

		{
			name: "size 15 has 8 leaves",
			args: args{
				size: 14 + 1,
			},
			leaves: 8,
		},
		{
			name: "size 11 has 7 leaves",
			args: args{
				size: 10 + 1,
			},
			leaves: 7,
		},

		{
			name: "invalid size 12 has 7 leaves",
			args: args{
				size: 11 + 1,
			},
			leaves: 7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LeafCount(tt.args.size); got != tt.leaves {
				t.Errorf("LeafCount() = %v, want %v", got, tt.leaves)
			}
		})
	}
}

// TestLeafCountFirst26 this test exists to show the behavior when LeafCount is
// given invalid mmrSizes.  Essentially, it returns the result of the highest
// valid mmrSize <= the provided size. And this can make its behavior
// non-obvious when it is used with arbitrary sizes.
func TestLeafCountFirst26(t *testing.T) {

	// expectLeafCounts is expressed in binary to illustrate that the consecutive valid
	// values for the binary accumulator are precisely the leaves. Essentially,
	// when we add a leaf to an MMR, we are doing the binary carry operation.
	// This is why we get 'smears' of leaf counts for invalid mmrSizes. The
	// correspond to not having fully "carried the bit addition". When we run
	// the PeaksBitmap (which is how LeafCount works), on the intermediate
	// values, it terminates at the last valid mmrSize.
	expectLeafCounts := []uint64{
		// 	1, 1, 2, 3, 3, 3, 4, 5, 5, 6, 7, 7, 7, 7, 8, 9, 9, 10, 11, 11, 11, 12, 13, 13, 14, 15,
		0b1, 0b1, 0b10, 0b11, 0b11, 0b11, 0b100, 0b101, 0b101, 0b110, 0b111, 0b111, 0b111, 0b111,
		0b1000, 0b1001, 0b1001, 0b1010, 0b1011, 0b1011, 0b1011, 0b1100, 0b1101, 0b1101, 0b1110, 0b1111,
	}

	var leafCounts []uint64

	for mmrIndex := uint64(0); mmrIndex < 26; mmrIndex++ {
		// i+1 converts from mmrIndex to mmrSize
		mmrSize := mmrIndex + 1
		got := LeafCount(mmrSize)
		assert.Equal(t, got, expectLeafCounts[mmrIndex])
		leafCounts = append(leafCounts, got)
	}
	for i := range leafCounts {
		fmt.Printf("%04d, ", i+1)
	}
	fmt.Printf("\n")
	for _, l := range leafCounts {
		fmt.Printf("%04b, ", l)
	}

	fmt.Printf("\n")
}
