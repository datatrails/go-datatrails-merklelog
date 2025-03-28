package massifs

import (
	"reflect"
	"testing"
)

func TestPeakStackMap(t *testing.T) {
	type args struct {
		massifHeight uint8
		mmrIndex     uint64
	}
	tests := []struct {
		name string
		args args
		want map[uint64]int
	}{
		// Note that the mmrSize used here, is also the FirstLeaf + 1 of the
		// massif containing the peak stack.
		{"massifpeakstack_test:0", args{2, 0}, map[uint64]int{}},
		{"massifpeakstack_test:1", args{2, 3}, map[uint64]int{
			2: 0,
		}},
		{"massifpeakstack_test:2", args{2, 6}, map[uint64]int{
			6: 0,
		}},

		{"massifpeakstack_test:3", args{2, 9}, map[uint64]int{
			6: 0,
			9: 1,
		}},
		{"massifpeakstack_test:4", args{2, 14}, map[uint64]int{
			14: 0,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PeakStackMap(tt.args.massifHeight, tt.args.mmrIndex); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PeakStackMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
