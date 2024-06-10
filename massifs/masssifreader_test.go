package massifs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMassifIndexFromLeafIndex tests:
//
// Example MMR for test 1,2,3. Derived from: https://github.com/datatrails/epic-8120-scalable-proof-mechanisms/blob/main/mmr/forestrie-mmrblobs.md#blob-size
//
// |    6     |     13      |     21      | height: 3
// |   /  \   |   /    \    |    /  \     |
// |  2    5  |  9     12   |  17    20   | height: 2
// | / \ /  \ | / \   /  \  | /  \   / \  |
// |0   1 3  4|7   8 10   11|15  16 18  19| MMR INDICES height: 1
// |----------|-------------|-------------|
// |0   1 2  3|4   5 6    7 | 8   9 10  11| LEAF INDICES
// |----------|-------------|-------------|
// |     0    |      1      |      2      | MASSIF INDICES
// |----------|-------------|-------------|
//
// 1. a height of 3 and a leaf index of 3, returns a massif index of 0
// 2. a height of 3 and a leaf index of 10, returns a massif index of 2
// 3. a height of 3 and a leaf index of 4, returns a massif index of 1
// 4. a height of 5 and a leaf index of 25, returns a massif index of 1
// 5. a height of 16 and a leaf index of 965235, returns a massif index of 29
func TestMassifIndexFromLeafIndex(t *testing.T) {
	type args struct {
		massifHeight uint8
		leafIndex    uint64
	}
	tests := []struct {
		name     string
		args     args
		expected uint64
	}{
		{
			name: "height 2, leaf index 3",
			args: args{
				massifHeight: 3,
				leafIndex:    3,
			},
			expected: 0,
		},
		{
			name: "height 2, leaf index 10",
			args: args{
				massifHeight: 3,
				leafIndex:    10,
			},
			expected: 2,
		},
		{
			name: "height 2, leaf index 4",
			args: args{
				massifHeight: 3,
				leafIndex:    4,
			},
			expected: 1,
		},
		{
			name: "height 4, leaf index 25",
			args: args{
				massifHeight: 5,
				leafIndex:    25,
			},
			expected: 1,
		},
		{
			name: "height 4, leaf index 25",
			args: args{
				massifHeight: 16,
				leafIndex:    965235,
			},
			expected: 29,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := MassifIndexFromLeafIndex(test.args.massifHeight, test.args.leafIndex)

			assert.Equal(t, test.expected, actual)
		})
	}
}

// TestMassifIndexFromMMRIndex tests:
//
// Example MMR for test 1,2,3. Derived from: https://github.com/datatrails/epic-8120-scalable-proof-mechanisms/blob/main/mmr/forestrie-mmrblobs.md#blob-size
//
// |    6     |     13      |     21      | height: 3
// |   /  \   |   /    \    |    /  \     |
// |  2    5  |  9     12   |  17    20   | height: 2
// | / \ /  \ | / \   /  \  | /  \   / \  |
// |0   1 3  4|7   8 10   11|15  16 18  19| MMR INDICES height: 1
// |----------|-------------|-------------|
// |0   1 2  3|4   5 6    7 | 8   9 10  11| LEAF INDICES
// |----------|-------------|-------------|
// |     0    |      1      |      2      | MASSIF INDICES
// |----------|-------------|-------------|
//
// 1. a height of 3 and a mmr index of 4, returns a massif index of 0
// 2. a height of 3 and a mmr index of 15, returns a massif index of 2
// 3. a height of 3 and a mmr index of 10, returns a massif index of 1
// 4. a height of 3 and a mmr index of 11, returns a massif index of 1
// 5. a height of 3 and a mmr index of 14, returns a massif index of 1
// 6. a height of 5 and a mmr index of 33, returns a massif index of 1
// 7. a height of 5 and a mmr index of 70, returns a massif index of 1
// 8. a height of 3 and a mmr index of 12, returns a massif index of 1
// 9. a height of 5 and a mmr index of 72, returns a massif index of 2
// 10. a height of 14 and a mmr index of 16382, returns a massif index of 0
// 11. a height of 14 and a mmr index of 16383, returns a massif index of 1
// 12. a height of 14 and a mmr index of 999418, retruns a massif index of 60
// 13. a height of 14 and a mmr index of 999419, retruns a massif index of 61
func TestMassifIndexFromMMRIndex(t *testing.T) {
	type args struct {
		massifHeight uint8
		mmrIndex     uint64
	}
	tests := []struct {
		name     string
		args     args
		expected uint64
		err      error
	}{
		{
			name: "height 3, mmr index 4",
			args: args{
				massifHeight: 3,
				mmrIndex:     4,
			},
			expected: 0,
		},
		{
			name: "height 3, mmr index 15",
			args: args{
				massifHeight: 3,
				mmrIndex:     15,
			},
			expected: 2,
		},
		{
			name: "height 3, mmr index 10",
			args: args{
				massifHeight: 3,
				mmrIndex:     10,
			},
			expected: 1,
		},
		{
			name: "height 3, mmr index 11",
			args: args{
				massifHeight: 3,
				mmrIndex:     11,
			},
			expected: 1,
		},
		{
			name: "height 3, mmr index 14",
			args: args{
				massifHeight: 3,
				mmrIndex:     14,
			},
			expected: 1,
		},
		{
			name: "height 5, mmr index 32",
			args: args{
				massifHeight: 5,
				mmrIndex:     32,
			},
			expected: 1,
		},
		{
			name: "height 5, mmr index 70",
			args: args{
				massifHeight: 5,
				mmrIndex:     70,
			},
			expected: 2,
		},
		{
			name: "height 5, mmr index 70",
			args: args{
				massifHeight: 5,
				mmrIndex:     70,
			},
			expected: 2,
		},
		{
			name: "height 3, mmr index 12",
			args: args{
				massifHeight: 3,
				mmrIndex:     12,
			},
			expected: 1,
		},
		{
			name: "height 5, mmr index 72",
			args: args{
				massifHeight: 5,
				mmrIndex:     72,
			},
			expected: 2,
		},
		{
			name: "height 14, mmr index 16382",
			args: args{
				massifHeight: 14,
				mmrIndex:     16382,
			},
			expected: 0,
		},
		{
			name: "height 14, mmr index 16383",
			args: args{
				massifHeight: 14,
				mmrIndex:     16383,
			},
			expected: 1,
		},
		{
			name: "height 14, mmr index 999418",
			args: args{
				massifHeight: 14,
				mmrIndex:     999418,
			},
			expected: 60,
		},
		{
			name: "height 14, mmr index 999419",
			args: args{
				massifHeight: 14,
				mmrIndex:     999419,
			},
			expected: 61,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := MassifIndexFromMMRIndex(test.args.massifHeight, test.args.mmrIndex)

			assert.Equal(t, test.expected, actual)
		})
	}
}
