package mmr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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
		//  0  1 .2 .3 .4  5 .6 .7  8  9 10 11 12 13 14 15 16 17  18  19  20  21  22  23
		//  1 .2 .3 .4  5 .6 .7  8  9 10 11 12 13 14 15 16 17  18  19  20  21  22  23  24
		// 	1, 1, 2, 3, 3, 3, 4, 5, 5, 6, 7, 7, 7, 7, 8, 9, 9, 10, 11, 11, 11, 12, 13, 13, 14, 15,
		// 	0, 0, 1, 2, 2, 2, 3, 4, 4, 5, 6, 6, 6, 6, 7, 8, 8,  9, 10, 10, 10, 11, 12, 12, 13, 14,
		0b1, 0b1, 0b10, 0b11, 0b11, 0b11, 0b100, 0b101, 0b101, 0b110, 0b111, 0b111, 0b111, 0b111,
		0b1000, 0b1001, 0b1001, 0b1010, 0b1011, 0b1011, 0b1011, 0b1100, 0b1101, 0b1101, 0b1110, 0b1111,
	}

	var leafCounts []uint64

	// for mmrIndex := uint64(0); mmrIndex < 26; mmrIndex++ {
	for mmrIndex := range uint64(38) {
		// i+1 converts from mmrIndex to mmrSize
		mmrSize := mmrIndex + 1
		got := LeafCount(mmrSize)
		if len(expectLeafCounts) > int(mmrIndex) {
			assert.Equal(t, got, expectLeafCounts[mmrIndex])
		}
		leafCounts = append(leafCounts, got)
	}
	for i := range leafCounts {
		fmt.Printf("%05d, ", i)
	}
	fmt.Printf("\n")
	for _, l := range leafCounts {
		fmt.Printf("%05b, ", l)
	}
	fmt.Printf("\n")
	for _, l := range leafCounts {
		fmt.Printf("%05d, ", l)
	}
	fmt.Printf("\n")
	for _, l := range leafCounts {
		fmt.Printf("%05d, ", l-1)
	}
	fmt.Printf("\n")

}

func TestFirstMMRSize(t *testing.T) {

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

	// This test iterates through a sequential range of mmrIndices, the test values are the sizes we expect.
	tests := []uint64{
		//0 1 2  3  4  5  6  7   8   9  10  11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25
		1, 3, 3, 4, 7, 7, 7, 8, 10, 10, 11, 15, 15, 15, 15, 16, 18, 18, 19, 22, 22, 22, 23, 25, 25, 26,
	}

	for i, want := range tests {
		t.Run(fmt.Sprintf("mmrIndex %d", i), func(t *testing.T) {
			got := FirstMMRSize(uint64(i))
			if got != want {
				t.Errorf("FirstMMRSize() = %v, want %v", got, want)
			}
			// this is to illustrate the confusion that arises from using LeafCount directly on arbitrary indices
			leavesFromIndex := LeafCount(uint64(i) + 1)
			leavesFromSize := LeafCount(got)
			fmt.Printf("i=%02d, LeafCount(i+1) = %d, LeafCount(FirstMMRSize(i)) = %d\n", i, leavesFromIndex, leavesFromSize)
		})
	}
}

func TestLeafIndex(t *testing.T) {

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
	// 0   0   1 2   3  4   5  6   7  8   9 10  11 12  13   14

	// This test iterates through a sequential range of mmrIndices, the test values are the sizes we expect.
	tests := []uint64{
		//0 1 2  3  4  5  6  7  8  9 10 11 12 13 14 15 16 17  18  19  20  21 22, 23, 24, 25
		0, 1, 1, 2, 3, 3, 3, 4, 5, 5, 6, 7, 7, 7, 7, 8, 9, 9, 10, 11, 11, 11, 12, 13, 13, 14,
	}

	for i, want := range tests {
		t.Run(fmt.Sprintf("mmrIndex %d", i), func(t *testing.T) {
			got := LeafIndex(uint64(i))
			if got != want {
				t.Errorf("LeafIndex(%d) = %d, want %d", i, got, want)
			}
			// this is to illustrate the confusion that arises from using LeafCount directly on arbitrary indices
			leavesFromIndex := LeafCount(uint64(i) + 1)
			fmt.Printf("i=%02d, LeafCount(i+1) = %02d, LeafIndex(i) = %02d\n", i, leavesFromIndex, got)
		})
	}

}
