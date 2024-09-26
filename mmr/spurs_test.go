package mmr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpurSum(t *testing.T) {
	type args struct {
		height uint64
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{"H=5", args{5}, 26},
		{"H=4", args{4}, 11},
		{"H=3", args{3}, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SpurSumHeight(tt.args.height); got != tt.want {
				t.Errorf("SpurSum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTreeIndexOld(t *testing.T) {

	treeIndices := []uint64{0, 1, 3, 4, 7, 8, 10, 11, 15, 16, 18, 19, 22, 23, 25, 26}
	for iLeaf, want := range treeIndices {
		t.Run(fmt.Sprintf("%d -> %d", iLeaf, want), func(t *testing.T) {
			var got uint64
			if got = TreeIndexOld(uint64(iLeaf)); got != want {
				t.Errorf("TreeIndexOld() = %v, want %v", got, treeIndices[iLeaf])
			}
			fmt.Printf("%d -> %d\n", iLeaf, want)
		})
	}
}

func TestSpurHeightLeaf(t *testing.T) {

	heights := []uint64{
		//	3            14                             29
		//	           /    \                                \
		//	          /      \                     /          \
		//	         /        \                   /            \
		//	2      6 .      .  13                21            28
		//	      /   \       /   \             / . \         /   \
		//	1    2     5     9     12         17     20     24     27
		//	    / \  /  \   / \    /  \      /  \   / \     / \ . ./ \
		//	   0   1 3   4  7   8 10   11  15   16 18  19  22  23 25  26 MMR INDICES
		//	   0 . 1 2 . 3  4   5  6    7 . 8 .  9 10  11  12  13 14  15 LEAF INDICES

		0, // 0
		1, // 1
		0, // 2
		2, // 3
		0, // 4
		1, // 5
		0, // 6
		3, // 7
		0, // 8
		1, // 9
		0, // 10
		2, // 11
		0, // 12
		1, // 13
		0, // 14
		4, // 15
		0, // 16
		1, // 17
	}
	for i, want := range heights {
		t.Run(fmt.Sprintf("%d -> %d", i, want), func(t *testing.T) {
			delta := SpurHeightLeaf(uint64(i))
			assert.Equal(t, delta, want)
		})
	}
}

func TestLeafMinusSpurSum(t *testing.T) {
	sums := []uint64{
		//	3            14                             29
		//	           /    \                                \
		//	          /      \                     /          \
		//	         /        \                   /            \
		//	2      6 .      .  13                21            28
		//	      /   \       /   \             / . \         /   \
		//	1    2     5     9     12         17     20     24     27
		//	    / \  /  \   / \    /  \      /  \   / \     / \ . ./ \
		//	   0   1 3   4  7   8 10   11  15   16 18  19  22  23 25  26 MMR INDICES
		//	   0 . 1 2 . 3  4   5  6    7 . 8 .  9 10  11  12  13 14  15 LEAF INDICES

		0, // 0
		1, // 1
		1, // 2
		2, // 3
		1, // 4
		2, // 5
		2, // 6
		3, // 7
		1, // 8
		2, // 9
		2, // 10
		3, // 11
		2, // 12
		3, // 13
		3, // 14
		4, // 15
		1, // 16
		2, // 17
	}
	for iLeaf, want := range sums {
		t.Run(fmt.Sprintf("%d -> %d", iLeaf, want), func(t *testing.T) {
			sum := LeafMinusSpurSum(uint64(iLeaf))
			assert.Equal(t, sum, want)

			// Test that the stack like property is maintained
			top := uint64(0)
			for i := uint64(0); i < uint64(iLeaf); i++ {

				delta := SpurHeightLeaf(i)
				top -= delta // pop
				top += 1     // push
				// fmt.Printf("%02d: %d", i, a)
				// ancestors := mmr.LeftAncestors(mmr.MMRIndex(i))
				// fmt.Printf("%02d: %d %d %d: ", i, top+delta, delta, top)
				// for _, a := range ancestors {
				// 	fmt.Printf("%d ", a)
				// }
				// fmt.Printf("\n")
			}
			assert.Equal(t, top, sum)

		})
	}
}
