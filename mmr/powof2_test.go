package mmr

import "testing"

func Test_isPow2(t *testing.T) {
	type args struct {
		size uint
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"16 is a power of two",
			args{
				16,
			},
			true,
		},
		{
			"zero is not a power of two",
			args{
				0,
			},
			false,
		},
		{
			"1 is a power of two",
			args{
				1,
			},
			true,
		},
		{
			"17 is not a power of two (first bit is set, edge case)",
			args{
				17,
			},
			false,
		},
		{
			"18 is not a power of two",
			args{
				18,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPow2(tt.args.size); got != tt.want {
				t.Errorf("isPow2() = %v, want %v", got, tt.want)
			}
		})
	}
}
