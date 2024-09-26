package massifs

import (
	"errors"
	"testing"

	"github.com/datatrails/go-datatrails-merklelog/mmr"
	"github.com/stretchr/testify/assert"
)

func TestMassifLogEntries(t *testing.T) {

	// Working with height 1 massifs and the following overall MMR
	//
	//  4                        30
	//
	//
	//               14                        29
	//	3           /  \                      /   \
	//	           /    \                    /     \
	//	          /      \                  /       \
	//	         /        \                /         \
	//	2      6 .      .  13             21          28
	//	      /   \       /   \          /  \        /   \
	//	1    2  |  5  |  9  |  12   |  17  | 20   | 24   | 27   |  --- massif tree line massif height = 1
	//	    / \ |/  \ | / \ |  /  \ | /  \ | / \  | / \  | / \  |
	//	   0   1|3   4|7   8|10   11|15  16|18  19|22  23|25  26| MMR INDICES
	//     -----|-----|-----|-------|------|------|------|------|
	//	   0 . 1|2 . 3|4   5| 6    7| 8   9|10  11|12  13|14  15| LEAF INDICES
	//     -----|-----|-----|-------|------|------|------|------|
	//       0  |  1  |  2  |  3    |   4  |   5  |   6  |   7  | MASSIF INDICES
	//     -----|-----|-----|-------|------|------|------|------|

	// I think it is sufficient to do this just for one massifHeight, but we could extend this approach.
	massifHeight := uint64(2) // each masif has 2 leaves and 3 nodes + spur
	massifNodeCount := uint64(2<<massifHeight - 1)
	massifLeafCount := (massifNodeCount + 1) / 2
	trieDataSize := 64 * massifLeafCount

	tests := []struct {
		name        string
		startOffset int64
		endOffset   int64
		want        uint64
	}{
		{
			name:        "underflow and over flow",
			startOffset: -1,
			endOffset:   -1,
		},
		{
			name:        "full but correct range",
			startOffset: 0,
			endOffset:   0,
		},
	}

	// This aligns with the constants in logformat.go. it is purposefully defined
	// here explicitly so we reconsider this test if the constants change.
	// ValueBytes * 8 + IndexHeaderBytes + TrieEntryBytes * leafCount
	fixedHeaderEnd := uint64(32*7 + 32)

	// these are mostly not worth their own unit tests and are very easy to
	// check here. and having these guards here model the co-dependence on the
	// primitives for finding the right bits of the format.
	gotFixedHeaderEnd := FixedHeaderEnd()
	assert.Equal(t, fixedHeaderEnd, gotFixedHeaderEnd)

	gotTrieDataSize := TrieDataSize(uint8(massifHeight))
	assert.Equal(t, trieDataSize, gotTrieDataSize)

	trieDataEnd := int64(fixedHeaderEnd + 32 + trieDataSize)

	gotTrieDataEnd := TrieDataEnd(uint8(massifHeight))
	assert.Equal(t, trieDataEnd, int64(gotTrieDataEnd))

	// massifIndex: peakStackLen
	stackLens := map[uint64]int64{
		0: 0,
		1: 1,
		2: 1,
		3: 2,
		4: 1,
		5: 2,
		6: 2,
		7: 3,
		8: 1,
	}

	// This tests that all *valid* mmr data lengths
	/* */
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			for massifIndex := uint64(0); massifIndex < 8; massifIndex++ {

				// each blob has a varying (but computable) number of mmr nodes.
				// massif height just determines how many *leaves* are in a full
				// blob.. and the leaf count is constant for all blobs. The back
				// fill nodes required for each blobs 'last' leaf varies.
				firstIndex := mmr.MMRIndex(massifLeafCount * massifIndex)
				lastIndex := mmr.MMRIndex(massifLeafCount*(massifIndex+1)) - 1
				expectNodeCount := lastIndex - firstIndex + 1

				// for each massif try a range of log sizes. the sizes can be invalid if the startOffset or endOffset are negative.
				for massifNodeIndex := 0 + tt.startOffset; massifNodeIndex < int64(expectNodeCount)-tt.endOffset; massifNodeIndex++ {

					// we use a lookup table to define the expected peak stack
					// lengths so this test doesn't overly rely on the
					// underlying primitives we want to cover.
					dataLen := int(trieDataEnd + stackLens[massifIndex]*32 + massifNodeIndex*32)

					nodeCount, err := MassifLogEntries(dataLen, massifIndex, uint8(massifHeight))
					if massifNodeIndex < 0 {
						if !errors.Is(err, ErrMassifDataLengthInvalid) {
							t.Errorf("expected err %v, got %v", ErrBeforeFirstLeaf, err)
						}
					} else {
						assert.NoError(t, err)

						assert.Equal(t, nodeCount, uint64(massifNodeIndex))
					}
				}
			}
			// if end offset or start offset are negative we should get an error
		})
	}
}

func TestTrieDataEntryCount(t *testing.T) {
	tests := []struct {
		name         string
		massifHeight uint8
		want         uint64
	}{
		{massifHeight: 0, want: 1},
		{massifHeight: 1, want: 2},
		{massifHeight: 2, want: 4},
		{massifHeight: 3, want: 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TrieDataEntryCount(tt.massifHeight); got != tt.want {
				t.Errorf("TrieDataEntryCount() = %v, want %v", got, tt.want)
			}
		})
	}
}
