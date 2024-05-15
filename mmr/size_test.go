package mmr

import (
	"fmt"
	"testing"
)

func TestHeightIndexSize(t *testing.T) {
	type args struct {
		heightIndex uint64
	}
	tests := []struct {
		args args
		want uint64
	}{
		{args{0}, 1},
		{args{1}, 3},
		{args{2}, 7},
		{args{3}, 15},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("heightIndex %d = size %d", tt.args.heightIndex, tt.want), func(t *testing.T) {
			if got := HeightIndexSize(tt.args.heightIndex); got != tt.want {
				t.Errorf("HeightSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeightIndex(t *testing.T) {
	type args struct {
		heightIndex uint64
	}
	tests := []struct {
		args args
		want uint64
	}{
		{args{0}, 0},
		{args{1}, 2},
		{args{2}, 6},
		{args{3}, 14},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("heightIndex %d = size %d", tt.args.heightIndex, tt.want), func(t *testing.T) {
			if got := HeightMaxIndex(tt.args.heightIndex); got != tt.want {
				t.Errorf("HeightSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeightSize(t *testing.T) {
	type args struct {
		heightIndex uint64
	}
	tests := []struct {
		args args
		want uint64
	}{
		{args{1}, 1},
		{args{2}, 3},
		{args{3}, 7},
		{args{4}, 15},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("heightIndex %d = size %d", tt.args.heightIndex, tt.want), func(t *testing.T) {
			if got := HeightSize(tt.args.heightIndex); got != tt.want {
				t.Errorf("HeightSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
