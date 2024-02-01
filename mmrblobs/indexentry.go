package mmrblobs

import (
	"encoding/binary"
	"errors"

	"github.com/google/uuid"
)

const (
	IndexEntryBytes     = 32 * 2
	KeyBitSizeLogBase2  = 8
	KeyByteSizeLogBase2 = 5

	EventIDFirst     = 0
	EventIDEnd       = EventIDFirst + 16
	SnowflakeIdFirst = 24
	SnowflakeIdEnd   = SnowflakeIdFirst + 8
	AssetIDFirst     = SnowflakeIdEnd
	AssetIDEnd       = AssetIDFirst + 16
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
	assetId uuid.UUID, eventId uuid.UUID, snowflakeId uint64,
) []byte {
	index := [IndexEntryBytes]byte{}

	SetIndexEntry(index[:], 0, assetId, eventId, snowflakeId)
	return index[:]
}

// SetIndexEntry populates the mmr blob index entry at the provided data offset
//
// | 0 -   127 | 128 - 185| 184 - 191        | 192 -  255 |
// | event uuid| reserved | reserved (epoch) | snowflakeid|
// | 0  -   15 | 16 -   22|     23           | 24   -   31|
// |     16    |     7    |     1            |      8     |
// | asset uuid|        reserved        |
// | 256 -  384| 384 -           -  512 |
// |     16    |           16           |
func SetIndexEntry(
	data []byte, offset uint64,
	assetId uuid.UUID, eventId uuid.UUID, snowflakeId uint64,
) {
	copy(data[offset+EventIDFirst:offset+EventIDEnd], eventId[:])
	copy(data[offset+AssetIDFirst:offset+AssetIDEnd], assetId[:])

	binary.BigEndian.PutUint64(data[offset+SnowflakeIdFirst:offset+SnowflakeIdEnd], snowflakeId)
}
