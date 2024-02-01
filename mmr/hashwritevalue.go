package mmr

import (
	"encoding/binary"
	"hash"
)

// hashWriteUInt64 writes a uint64 to a hasher in bigendian layout - most
// significant byte at lowest address/storage location
func hashWriteUint64(hasher hash.Hash, value uint64) {
	b := [8]byte{}
	binary.BigEndian.PutUint64(b[:], value)
	hasher.Write(b[:])
}
