package mmr

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
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
		peakBits    uint64
		heightIndex uint64
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PeakIndex(tt.args.peakBits, tt.args.heightIndex); got != tt.want {
				t.Errorf("PeakIndex() = %v, want %v", got, tt.want)
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
