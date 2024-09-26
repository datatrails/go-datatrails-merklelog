package mmr

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeaks(t *testing.T) {
	type args struct {
		mmrSize uint64
	}
	tests := []struct {
		name string
		args args
		want []uint64
	}{

		{"size 11 gives three peaks", args{11}, []uint64{7, 10, 11}},
		{"size 26 gives 4 peaks", args{26}, []uint64{15, 22, 25, 26}},
		{"size 10 gives two peaks", args{10}, []uint64{7, 10}},
		{"size 13, which is invalid because it should have been perfectly filled, gives nil", args{13}, nil},
		{"size 15, which is perfectly filled, gives a single peak", args{15}, []uint64{15}},
		{"size 18 gives two peaks", args{18}, []uint64{15, 18}},
		{"size 22 gives two peaks", args{22}, []uint64{15, 22}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Peaks(tt.args.mmrSize); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Peaks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPeaksKAT_MMR39(t *testing.T) {
	tests := []struct {
		mmrSize uint64
		want    []uint64
	}{
		{1, []uint64{1}},
		{3, []uint64{3}},
		{4, []uint64{3, 4}},
		{7, []uint64{7}},
		{8, []uint64{7, 8}},
		{10, []uint64{7, 10}},
		{11, []uint64{7, 10, 11}},
		{15, []uint64{15}},
		{16, []uint64{15, 16}},
		{18, []uint64{15, 18}},
		{19, []uint64{15, 18, 19}},
		{22, []uint64{15, 22}},
		{23, []uint64{15, 22, 23}},
		{25, []uint64{15, 22, 25}},
		{26, []uint64{15, 22, 25, 26}},
		{31, []uint64{31}},
		{32, []uint64{31, 32}},
		{34, []uint64{31, 34}},
		{35, []uint64{31, 34, 35}},
		{38, []uint64{31, 38}},
		{39, []uint64{31, 38, 39}},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.mmrSize), func(t *testing.T) {
			if got := Peaks(tt.mmrSize); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Peaks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPeakHashesKAT_MMR39(t *testing.T) {
	tests := []struct {
		mmrSize uint64
		want    []string
	}{
		{1, []string{"af5570f5a1810b7af78caf4bc70a660f0df51e42baf91d4de5b2328de0e83dfc"}},
		{3, []string{"ad104051c516812ea5874ca3ff06d0258303623d04307c41ec80a7a18b332ef8"}},
		{4, []string{"ad104051c516812ea5874ca3ff06d0258303623d04307c41ec80a7a18b332ef8", "d5688a52d55a02ec4aea5ec1eadfffe1c9e0ee6a4ddbe2377f98326d42dfc975"}},
		{7, []string{"827f3213c1de0d4c6277caccc1eeca325e45dfe2c65adce1943774218db61f88"}},
		{8, []string{"827f3213c1de0d4c6277caccc1eeca325e45dfe2c65adce1943774218db61f88", "a3eb8db89fc5123ccfd49585059f292bc40a1c0d550b860f24f84efb4760fbf2"}},
		{10, []string{"827f3213c1de0d4c6277caccc1eeca325e45dfe2c65adce1943774218db61f88", "b8faf5f748f149b04018491a51334499fd8b6060c42a835f361fa9665562d12d"}},
		{11, []string{"827f3213c1de0d4c6277caccc1eeca325e45dfe2c65adce1943774218db61f88", "b8faf5f748f149b04018491a51334499fd8b6060c42a835f361fa9665562d12d", "8d85f8467240628a94819b26bee26e3a9b2804334c63482deacec8d64ab4e1e7"}},
		{15, []string{"78b2b4162eb2c58b229288bbcb5b7d97c7a1154eed3161905fb0f180eba6f112"}},
		{16, []string{"78b2b4162eb2c58b229288bbcb5b7d97c7a1154eed3161905fb0f180eba6f112", "e66c57014a6156061ae669809ec5d735e484e8fcfd540e110c9b04f84c0b4504"}},
		{18, []string{"78b2b4162eb2c58b229288bbcb5b7d97c7a1154eed3161905fb0f180eba6f112", "f4a0db79de0fee128fbe95ecf3509646203909dc447ae911aa29416bf6fcba21"}},
		{19, []string{"78b2b4162eb2c58b229288bbcb5b7d97c7a1154eed3161905fb0f180eba6f112", "f4a0db79de0fee128fbe95ecf3509646203909dc447ae911aa29416bf6fcba21", "5bc67471c189d78c76461dcab6141a733bdab3799d1d69e0c419119c92e82b3d"}},
		{22, []string{"78b2b4162eb2c58b229288bbcb5b7d97c7a1154eed3161905fb0f180eba6f112", "61b3ff808934301578c9ed7402e3dd7dfe98b630acdf26d1fd2698a3c4a22710"}},
		{23, []string{"78b2b4162eb2c58b229288bbcb5b7d97c7a1154eed3161905fb0f180eba6f112", "61b3ff808934301578c9ed7402e3dd7dfe98b630acdf26d1fd2698a3c4a22710", "7a42e3892368f826928202014a6ca95a3d8d846df25088da80018663edf96b1c"}},
		{25, []string{"78b2b4162eb2c58b229288bbcb5b7d97c7a1154eed3161905fb0f180eba6f112", "61b3ff808934301578c9ed7402e3dd7dfe98b630acdf26d1fd2698a3c4a22710", "dd7efba5f1824103f1fa820a5c9e6cd90a82cf123d88bd035c7e5da0aba8a9ae"}},
		{26, []string{"78b2b4162eb2c58b229288bbcb5b7d97c7a1154eed3161905fb0f180eba6f112", "61b3ff808934301578c9ed7402e3dd7dfe98b630acdf26d1fd2698a3c4a22710", "dd7efba5f1824103f1fa820a5c9e6cd90a82cf123d88bd035c7e5da0aba8a9ae", "561f627b4213258dc8863498bb9b07c904c3c65a78c1a36bca329154d1ded213"}},
		{31, []string{"d4fb5649422ff2eaf7b1c0b851585a8cfd14fb08ce11addb30075a96309582a7"}},
		{32, []string{"d4fb5649422ff2eaf7b1c0b851585a8cfd14fb08ce11addb30075a96309582a7", "1664a6e0ea12d234b4911d011800bb0f8c1101a0f9a49a91ee6e2493e34d8e7b"}},
		{34, []string{"d4fb5649422ff2eaf7b1c0b851585a8cfd14fb08ce11addb30075a96309582a7", "0c9f36783b5929d43c97fe4b170d12137e6950ef1b3a8bd254b15bbacbfdee7f"}},
		{35, []string{"d4fb5649422ff2eaf7b1c0b851585a8cfd14fb08ce11addb30075a96309582a7", "0c9f36783b5929d43c97fe4b170d12137e6950ef1b3a8bd254b15bbacbfdee7f", "4d75f61869104baa4ccff5be73311be9bdd6cc31779301dfc699479403c8a786"}},
		{38, []string{"d4fb5649422ff2eaf7b1c0b851585a8cfd14fb08ce11addb30075a96309582a7", "6a169105dcc487dbbae5747a0fd9b1d33a40320cf91cf9a323579139e7ff72aa"}},
		{39, []string{"d4fb5649422ff2eaf7b1c0b851585a8cfd14fb08ce11addb30075a96309582a7", "6a169105dcc487dbbae5747a0fd9b1d33a40320cf91cf9a323579139e7ff72aa", "e9a5f5201eb3c3c856e0a224527af5ac7eb1767fb1aff9bd53ba41a60cde9785"}},
	}

	db := NewCanonicalTestDB(t)

	hexHashList := func(hashes [][]byte) []string {
		var hexes []string
		for _, b := range hashes {
			hexes = append(hexes, hex.EncodeToString(b))
		}
		return hexes
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.mmrSize), func(t *testing.T) {
			hashes, err := PeakHashes(db, tt.mmrSize)
			require.NoError(t, err)
			hexes := hexHashList(hashes)
			if !reflect.DeepEqual(hexes, tt.want) {
				t.Errorf("PeakHashes() = %v, want %v", hexes, tt.want)
			}
		})
	}
}

func TestAncestors(t *testing.T) {
	// POSITION TREE
	//
	//	3        \   15   massif 1 \ . massif 2
	//	          \/    \           \
	//	 massif 0 /\     \           |    'alpine zone' is above the massif tree line
	//	         /   \    \          |
	//	2 ..... 7.....|....14........|...... 22 ..... Masif Root Index identifies the massif root
	//	      /   \   |   /   \      |      /
	//	1    3     6  | 10     13    |    18     21
	//	    / \  /  \ | / \    /  \  |   /  \
	//	   1   2 4   5| 8   9 11   12| 16   17 19 20
	//	   | massif 0 |  massif 1 .  | massif 2 ....>
	//
	// INDEX TREE
	//	3        \   14   massif 1 \ . massif 2
	//	          \/    \           \
	//	 massif 0 /\     \           |    'alpine zone' is above the massif tree line
	//	         /   \    \          |
	//	2 ..... 6.....|....13........|...... 21 ..... Masif Root Index identifies the massif root
	//	      /   \   |   /   \      |      /
	//	1    2     5  |  9     12    |    18     20
	//	    / \  /  \ | / \    /  \  |   /  \
	//	   0   1 3   4| 7   8 10   11| 15   16 18 19
	//	   | massif 0 |  massif 1 .  | massif 2 ....>

	// lastFirst := uint64(0)

	massifHeight := uint64(2)
	massifSize := (2 << massifHeight) - 1
	fmt.Printf("height: %d, size: %d\n", massifHeight, massifSize)

	for i := uint64(0); i < 255; i++ {
		height := IndexHeight(i)
		if massifHeight != height {
			continue
		}
		ancestors := LeftAncestors(i + 1)
		if ancestors == nil {
			continue
		}
		// fmt.Printf("%03d %03d %d %d {", i+1, i+uint64(len(ancestors)/2)-lastFirst, height, len(ancestors)/2-1)
		//fmt.Printf("%d %d {", i+1, i+uint64(len(ancestors)/2)-lastFirst)

		massifCount := (2 << massifHeight) - 1 + len(ancestors)

		// fmt.Printf("%d %d {", i, i+uint64(len(ancestors))-lastFirst)
		fmt.Printf("%d %d: ", i, massifCount)
		for _, p := range ancestors {
			// fmt.Printf("%d - %d = %d", i, p, i-p)
			if (i - p) >= uint64(massifCount)-1 {
				fmt.Printf("%d = %d - %d, ", p, i, (i - p))
			}
		}
		//fmt.Printf("}[%d]\n", len(ancestors)/2)
		fmt.Printf("\n")
		// lastFirst = i + uint64(len(ancestors))
	}
	fmt.Printf("height: %d\n", massifHeight)
}

func TestTopHeight(t *testing.T) {
	type args struct {
		mmrSize uint64
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{"size 0 corner case", args{0}, 0},
		{"size 1 corner case", args{1}, 1},
		{"size 2", args{2}, 1},
		{"size 3", args{3}, 2},
		{"size 4, two peaks, single solo at i=3", args{4}, 2},
		{"size 5, three peaks, two solo at i=3, i=4", args{5}, 2},
		{"size 6, two perfect peaks,i=2, i=5 (note add does not ever leave the MMR in this state)", args{6}, 2},
		{"size 7, one perfect peaks at i=6", args{7}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TopHeight(tt.args.mmrSize)
			if got != tt.want {
				t.Errorf("HighestPos() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func topPeakLongHand(pos uint64) uint64 {
	top := uint64(1)
	for (top - 1) <= pos {
		top <<= 1
	}
	return (top >> 1) - 1
}

func TestTopPeak(t *testing.T) {
	for pos := uint64(1); pos <= 39; pos++ {
		t.Run(fmt.Sprintf("TopPeak(%d)", pos), func(t *testing.T) {
			want := topPeakLongHand(pos)
			x := 1<<(BitLength64(pos+1)-1) - 1
			fmt.Printf("%d %4b %4b %d\n", x, x, pos, want)
			if got := TopPeak(pos); got != want {
				t.Errorf("TopPeak(%d) = %v, want %v", pos, got, want)
			}
		})
	}
}
func TestPeaks2(t *testing.T) {
	for pos := uint64(1); pos <= 39; pos++ {
		t.Run(fmt.Sprintf("Peaks2(%d)", pos), func(t *testing.T) {
			fmt.Printf("Peaks2(mmrSize: %d):", pos)
			peaks := PeaksOld(pos)
			peaks2 := Peaks(pos)
			assert.Equal(t, peaks, peaks2)
			fmt.Printf(" %v", peaks)
			fmt.Printf("\n")
		})
	}
}
func TestPeakIndex(t *testing.T) {
	type args struct {
	}
	tests := []struct {
		mmrIndex    uint64
		proofLength int
		expected    int
	}{
		{0, 0, 0}, // degenerate case

		{2, 1, 0}, // 2 is perfect

		// note the form here is len(accumulator) - 1 - the bit index from the right (least significant) with the zero's removed
		// except for the perfect peaks which are always 0
		{3, 1, 2 - 1 - 1},
		{3, 0, 2 - 1 - 0},

		{6, 2, 0}, // 10. 6 is perfect

		{7, 2, 2 - 1 - 1},
		{7, 0, 2 - 1 - 0},

		{9, 2, 2 - 1 - 1}, // 110.
		{9, 1, 2 - 1 - 0},

		{10, 2, 3 - 1 - 2}, // 111
		{10, 1, 3 - 1 - 1}, // 111
		{10, 0, 3 - 1 - 0}, // 111

		{14, 3, 0}, // 1000. 14 is perfect

		{15, 3, 2 - 1 - 1}, // 1001
		{15, 0, 2 - 1 - 0}, // 1001

		{17, 3, 2 - 1 - 1}, // 1010
		{17, 1, 2 - 1 - 0}, // 1010

		{18, 3, 3 - 1 - 2}, // 1011
		{18, 1, 3 - 1 - 1}, // 1011
		{18, 0, 3 - 1 - 0}, // 1011

		{21, 3, 2 - 1 - 1}, // 1100
		{21, 2, 2 - 1 - 0}, // 1100

		{22, 3, 3 - 1 - 2}, // 1101
		{22, 2, 3 - 1 - 1}, // 1101
		{22, 0, 3 - 1 - 0}, // 1101

		{24, 3, 3 - 1 - 2}, // 1110
		{24, 2, 3 - 1 - 1}, // 1110
		{24, 1, 3 - 1 - 0}, // 1110

		{25, 3, 4 - 1 - 3}, // 1111
		{25, 2, 4 - 1 - 2}, // 1111
		{25, 1, 4 - 1 - 1}, // 1111
		{25, 0, 4 - 1 - 0}, // 1111

		{30, 4, 0}, // 10000 perfect

		{31, 4, 2 - 1 - 1}, // 10001
		{31, 0, 2 - 1 - 0},

		{33, 4, 2 - 1 - 1}, // 10010
		{33, 1, 2 - 1 - 0},

		{34, 4, 3 - 1 - 2}, // 10011
		{34, 1, 3 - 1 - 1}, // 10011
		{34, 0, 3 - 1 - 0}, // 10011

		{37, 4, 2 - 1 - 1}, // 10100
		{37, 2, 2 - 1 - 0}, // 10100

		{38, 4, 3 - 1 - 2}, // 10101
		{38, 2, 3 - 1 - 1}, // 10101
		{38, 0, 3 - 1 - 0}, // 10101

	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("MMR(%d), proof length %d, expected peak %d", tt.mmrIndex, tt.proofLength, tt.expected), func(t *testing.T) {

			peakBits := LeafCount(tt.mmrIndex + 1)
			if got := PeakIndex(peakBits, tt.proofLength); got != tt.expected {
				t.Errorf("PeakIndex() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPeaksBitmap(t *testing.T) {
	tests := []struct {
		mmrSize uint64
		want    uint64
	}{
		{mmrSize: 10, want: 6},
		{mmrSize: 1, want: 1},
		{mmrSize: 3, want: 2},
		{mmrSize: 4, want: 3},
		{mmrSize: 7, want: 4},
		{mmrSize: 8, want: 5},
		{mmrSize: 11, want: 7},
		{mmrSize: 15, want: 8},
		{mmrSize: 16, want: 9},
		{mmrSize: 18, want: 10},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("PeaksBitmap(%d)", tt.mmrSize), func(t *testing.T) {
			got := PeaksBitmap(tt.mmrSize)
			fmt.Printf("%02d %05b %05b %05b %02d\n", tt.mmrSize, tt.mmrSize, tt.mmrSize-1, got, got)
			if got != tt.want {
				t.Errorf("PeaksBitmap(%d) = %v, want %v", tt.mmrSize, got, tt.want)
			}
		})
	}
}
