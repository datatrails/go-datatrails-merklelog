package mmrblobs

import (
	"encoding/binary"
	"errors"
)

const (
	IndexEntryBytes     = 32 * 2
	KeyBitSizeLogBase2  = 8
	KeyByteSizeLogBase2 = 5

	EntryKeyRandomPrefixFirst = 0
	EntryKeyRandomPrefixEnd   = EntryKeyRandomPrefixFirst + 16
	SnowflakeIdFirst          = 24
	SnowflakeIdEnd            = SnowflakeIdFirst + 8
	ApplicationDataFirst      = SnowflakeIdEnd
	ApplicationDataEnd        = ApplicationDataFirst + 16
)

var (
	ErrIndexEntryBadSize = errors.New("log index size invalid")
)

// EmptyIndexEntry is a convenience method for unit tests that don't require a valid index entry
func EmptyIndexEntry() []byte {
	return make([]byte, IndexEntryBytes)
}

func SetIndexSnowflakeID(
	data []byte, offset uint64,
	snowflakeId uint64,
) {
	binary.BigEndian.PutUint64(data[offset+SnowflakeIdFirst:offset+SnowflakeIdEnd], snowflakeId)
}

func GetIndexSnowflakeID(
	data []byte, offset uint64,
) uint64 {
	return binary.BigEndian.Uint64(data[offset+SnowflakeIdFirst : offset+SnowflakeIdEnd])
}

// NewIndexEntry creates an index entry directly from the required components
func NewIndexEntry(
	randomPrefix16 []byte, snowflakeId uint64, appData16 []byte,
) []byte {
	index := [IndexEntryBytes]byte{}

	SetIndexEntry(index[:], 0, randomPrefix16, snowflakeId, appData16)
	return index[:]
}

// SetIndexEntry populates the mmr blob index entry at the provided data offset
//
// | 0 -   127 | 128 - 185| 184 - 191        | 192 -  255 |
// | event uuid| reserved | reserved (epoch) | snowflakeid|
// | 0  -   15 | 16 -   22|     23           | 24   -   31|
// |     16    |     7    |     1            |      8     |
// | application data     |     reserved                  |
// | 256          -  383  | 384                     - 512 |
// |     16               |                  16           |
func SetIndexEntry(
	data []byte, offset uint64,
	randomPrefix16 []byte, snowflakeId uint64, appData16 []byte,
) {
	copy(data[offset+EntryKeyRandomPrefixFirst:offset+EntryKeyRandomPrefixEnd], randomPrefix16[:])
	copy(data[offset+ApplicationDataFirst:offset+ApplicationDataEnd], appData16[:])

	binary.BigEndian.PutUint64(data[offset+SnowflakeIdFirst:offset+SnowflakeIdEnd], snowflakeId)
}
