package mmrblobs

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMassifFirstLeaf(t *testing.T) {
	type args struct {
		massifHeight uint8
		massifIndex  uint32
	}
	tests := []struct {
		args args
		want uint64
	}{
		{args{3, 0}, 0},
		{args{3, 1}, 7},
		{args{3, 2}, 15},
		{args{3, 3}, 22},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("(h, mi) = (%d,%d) = %d", tt.args.massifHeight, tt.args.massifIndex, tt.want), func(t *testing.T) {
			if got := MassifFirstLeaf(tt.args.massifHeight, tt.args.massifIndex); got != tt.want {
				t.Errorf("MassifFirstLeaf() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMassifStartKeyRoundTrip(t *testing.T) {
	type args struct {
		version      uint16
		epoch        uint32
		massifHeight uint8
		massifIndex  uint32
		firstIndex   uint64
	}
	tests := []struct {
		name string
		args args
	}{
		{"a", args{1, 2, 2, 2, 7}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodeMassifStart(tt.args.version, tt.args.epoch, tt.args.massifHeight, tt.args.massifIndex)
			encoded = append(encoded, make([]byte, 32)...)
			got := MassifStart{}
			err := got.UnmarshalBinary(encoded)
			assert.Nil(t, err)
			assert.Equal(t, got.Version, tt.args.version)
			assert.Equal(t, got.Epoch, tt.args.epoch)
			assert.Equal(t, got.MassifHeight, tt.args.massifHeight)
			assert.Equal(t, got.MassifIndex, tt.args.massifIndex)
			assert.Equal(t, got.FirstIndex, tt.args.firstIndex)
		})
	}
}
