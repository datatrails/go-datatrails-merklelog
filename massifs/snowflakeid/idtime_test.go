package snowflakeid

import "testing"

func TestIDMilliSplit(t *testing.T) {
	type args struct {
		id uint64
	}
	tests := []struct {
		name  string
		args  args
		want  uint64
		want1 uint32
	}{
		{"fully f'd", args{(1 << 64) - 1}, (1 << 40) - 1, 0xffffff},
		{"1 bits", args{(1 << 24) | (1 << 8) | 1}, 1, 257},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := IDMilliSplit(tt.args.id)
			if got != tt.want {
				t.Errorf("IDMilliSplit() got = %x, want %x", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("IDMilliSplit() got1 = %x, want %x", got1, tt.want1)
			}
		})
	}
}
