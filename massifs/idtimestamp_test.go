package massifs

import (
	"reflect"
	"testing"
)

func TestIDTimestampBytes(t *testing.T) {
	type args struct {
		id    uint64
		epoch uint8
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		// check the expected locations for the serialization given the big endian encoding
		{args: args{id: 1, epoch: 1}, want: []byte{1, 0, 0, 0, 0, 0, 0, 0, 1}},
		// Here, 1 shifted left 63 bit positions creates a low address byte of
		// 128 in the serialized big endian representation of the unsigned int
		// 64. And the epoch is 1. And we check the epoch lands in byte[0]
		{args: args{id: 1 << 63, epoch: 1}, want: []byte{1, 128, 0, 0, 0, 0, 0, 0, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IDTimestampBytes(tt.args.id, tt.args.epoch); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IDTimestampBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitIDTimestampBytes(t *testing.T) {
	type args struct {
		b []byte
	}
	tests := []struct {
		name    string
		args    args
		want    uint64
		want1   uint8
		wantErr bool
	}{
		// The values in the byte slice correspond to the serialization of the epoch prefixed id timestamp
		{args: args{b: []byte{1, 0, 0, 0, 0, 0, 0, 0, 1}}, want: 1, want1: 1},
		// Here, 1 shifted left 63 bit positions creates a low address byte of
		// 128 in the serialized big endian representation of the unsigned int
		// 64. And the epoch is 1.
		{args: args{b: []byte{1, 128, 0, 0, 0, 0, 0, 0, 0}}, want: 1 << 63, want1: 1},
		// In this test, we have two bytes of epoch data. There is no
		// fundamental reason we cant deal with that, we won't benefit from the
		// extra complexity unless we are still around 4000 years from now and
		// programming is still a thing.
		{args: args{b: []byte{1, 1, 128, 0, 0, 0, 0, 0, 0, 0}}, want: 0, want1: 0, wantErr: true},
		// this case is just a straight up, data to short. It is only 7 bytes
		{args: args{b: []byte{0, 1, 0, 0, 0, 0, 0}}, want: 0, want1: 0, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := SplitIDTimestampBytes(tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("SplitIDTimestampBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SplitIDTimestampBytes() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("SplitIDTimestampBytes() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
