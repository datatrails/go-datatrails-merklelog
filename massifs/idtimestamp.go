package massifs

// When we serialise the id timestamps for propagation outside the subsystem we
// encode it with the epoch included as a prefix. This file containes utilities
// for dealing safely with that.

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"
)

var (
	ErrIDTimestampBytesToShort = errors.New("not enough bytes to represent an id time stamp")
	ErrEpochToLarge            = errors.New("we only currently support an 8 bit epoch counter")
)

// IDTimestampToHex returns the hex encoding of the id timestamp with the epoch
// pre-pended.  The epoch is the count of times we have overflowed 40 bits
// worth of milliseconds since the standard unix epoch. This will be 1 until Jan
// 2038

// Returns:
//
// An 18 character byte string, with the epoch hex value in [0:2], and the
// idtimestamp encoded as a big endian 64bit value and converted to hex.
func IDTimestampToHex(id uint64, epoch uint8) string {
	b := IDTimestampBytes(id, epoch)
	return hex.EncodeToString(b)
}

// SplitIDTimestampHex accepts a hex encoded, and epoch prefixed, id timestamp string
// Returns:
//
//	The 64 bit idtimestamp, the epoch or a non nil error
//
// Note: See IDTimestampBytes for description of the epoch
func SplitIDTimestampHex(id string) (uint64, uint8, error) {

	id = strings.TrimPrefix(id, "0x")

	b, err := hex.DecodeString(id)
	if err != nil {
		return 0, 0, err
	}
	return SplitIDTimestampBytes(b)
}

// IDTimestampBytes returns the serialization the id timestamp with the epoch
// pre-pended.  The epoch is the count of times we have overflowed 40 bits
// worth of milliseconds since the standard unix epoch. This will be 1 until Jan
// 2038
// Returns:
//
// A 9 byte slice, with the epoch in byte [0], and the idtimestamp encoded as a
// big endian 64bit value and stored in byte's [1:9]
func IDTimestampBytes(id uint64, epoch uint8) []byte {
	b := make([]byte, 8+1)
	b[0] = epoch
	binary.BigEndian.PutUint64(b[1:], id)
	return b
}

// SplitIDTimestampBytes accepts a serialized id timestamp, with the epoch prefixed.
// Returns:
//
//	The 64 bit idtimestamp, the epoch or a non nil error
//
// Note: See IDTimestampBytes for description of the epoch
func SplitIDTimestampBytes(b []byte) (uint64, uint8, error) {
	if len(b) < 8 {
		return 0, 0, ErrIDTimestampBytesToShort
	}
	id := binary.BigEndian.Uint64(b[1:])
	if len(b) > 9 {
		return 0, 0, ErrEpochToLarge
	}
	return id, b[0], nil
}
