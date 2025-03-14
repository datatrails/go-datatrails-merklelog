package mmr

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPosPeaks(t *testing.T) {
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
			if got := PosPeaks(tt.args.mmrSize); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PosPeaks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPeaks(t *testing.T) {
	type args struct {
		mmrIndex uint64
	}
	tests := []struct {
		name string
		args args
		want []uint64
	}{

		{"complete mmr(index 123) gives two peaks", args{FirstMMRSize(123)}, []uint64{126, 127}},
		{"index 123 gives nil", args{123}, []uint64(nil)},
		{"complete index 11 gives three peaks", args{10}, []uint64{6, 9, 10}},
		{"complete index 26 gives 4 peaks", args{25}, []uint64{14, 21, 24, 25}},
		{"complete index 9 gives two peaks", args{9}, []uint64{6, 9}},
		{"complete index 12, which is invalid because it should have been perfectly filled, gives nil", args{13}, nil},
		{"complete index 14, which is perfectly filled, gives a single peak", args{14}, []uint64{14}},
		{"complete index 17 gives two peaks", args{17}, []uint64{14, 17}},
		{"complete index 21 gives two peaks", args{21}, []uint64{14, 21}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Peaks(tt.args.mmrIndex); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Peaks() = %v, want %v", got, tt.want)
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

	for i := range uint64(255) {
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
		mmrIndex uint64
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		//	2       6
		//	      /   \
		//	1    2     5     9
		//	    / \  /  \   / \
		//	0  0   1 3   4 7   8 10

		{"complete index 0 corner case", args{0}, 0},
		{"complete index 2", args{2}, 1},
		{"complete index 3, two peaks, single solo at i=3", args{3}, 1},
		{"         index 4, three peaks, two solo at i=3, i=4", args{4}, 1},
		{"         index 5, two perfect peaks,i=2, i=5", args{5}, 1},
		{"complete index 7, one perfect peaks at i=6", args{6}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TopHeight(tt.args.mmrIndex)
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
	for i := range uint64(39) {
		t.Run(fmt.Sprintf("TopPeak(%d)", i), func(t *testing.T) {
			want := topPeakLongHand(i+1) - 1
			x := 1<<(BitLength64(i+1)-1) - 2
			fmt.Printf("%d %4b %4b %d\n", x, x, i, want)
			if got := TopPeak(i); got != want {
				t.Errorf("TopPeak(%d) = %v, want %v", i, got, want)
			}
		})
	}
}
func TestPeaks2(t *testing.T) {
	for pos := uint64(1); pos <= 39; pos++ {
		t.Run(fmt.Sprintf("Peaks2(%d)", pos), func(t *testing.T) {
			fmt.Printf("Peaks2(mmrSize: %d):", pos)
			peaks := PeaksOld(pos)
			peaks2 := PosPeaks(pos)
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
