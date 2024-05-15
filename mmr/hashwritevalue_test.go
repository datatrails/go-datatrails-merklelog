package mmr

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func decodeHex(t *testing.T, s string) []byte {
	v, err := hex.DecodeString(s)
	if err != nil {
		t.Errorf("could not hex decode %s", s)
	}
	return v
}

func Test_hashWriteUint64(t *testing.T) {

	type args struct {
		value uint64
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			"ff00000000000001", args{0xff00000000000001}, decodeHex(t, "0946348eb9631ac3fa6b8bedbbac750a06a6b13c8ca2c65a0f35914304b3b124"),
		},
		{
			"1", args{1}, decodeHex(t, "cd2662154e6d76b2b2b92e70c0cac3ccf534f9b74eb5b89819ec509083d00a50"),
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			hasher := sha256.New()
			HashWriteUint64(hasher, tt.args.value)
			got := hasher.Sum(nil)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("got %x, want %x", got, tt.want)
			}
		})
	}
}
