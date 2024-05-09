package massifs

// Massif blobs are strictly sized as multiples of 32 bytes in order to
// facilitate simple content independent arithmetic operations over the whole
// MMR.
//
// Knowing only the relative resource name of the blob (which includes its
// epoch), and the size of the blob all information necessary to place it in the
// overall MMR can be derived computationaly (and efficiently)
//
// The massifstart is a 32 byte field encoding the small amount of book keeping
// required in a blob to allow for efficient correctness checks. This field is
// followed by the root hashes from preceding blobs that will be necessary to
// complete the blob. These are maintained in a stack. Neither the stack length
// nor a mapping of the positions it contains are stored, all of this
// information is recovered computationally computed based on the blobs position
// in the MMR
import (
	"encoding/binary"
	"errors"

	"github.com/datatrails/forestrie/go-forestrie/mmr"
)

type KeyType uint8

const (
	KeyTypeApplicationContent KeyType = iota // this is the standard entry type, purposefully defined as 0
	_
	_
	_
	_
	_
	_
	_
	KeyTypeApplicationLast // first 8 types are reserved for the application
	// other entries are reserved for the MMR book keeping

	// KeyTypeMassifStart is the type for keys which correspond to massif blob
	// header values
	KeyTypeMassifStart
	KeyTypeMax = 255
)

const (

	// MassifStart layout
	//
	// .         | type| idtimestamp| reserved |  version | epoch  |massif height| massif i |
	// .         | 0   | 8        15|          |  21 - 22 | 23   26|27         27| 28 -  31 |
	// bytes     | 1   |     8      |          |      2   |    4   |      1      |     4    |
	//
	// Note this layout produces a sequentially valued key. The value is always
	// considered as a big endian large integer. Lexical ordering is defined
	// only for padded hex representations of the key value. The reserved zero
	// bytes can be used in later versions.  Because if we shift the version
	// field left, even without incrementing it, the resulting key is
	// numerically larger than all of those for previous versions

	MassifStartKeyLastIDFirstByte = 8
	MassifStartKeyLastIDSize      = 8 // 64 bits
	MassifStartKeyLastIDEnd       = MassifStartKeyLastIDFirstByte + MassifStartKeyLastIDSize
	// gap 16 - 21
	MassifStartKeyVersionFirstByte = 21
	MassifStartKeyVersionSize      = 2 // 16 bit
	MassifStartKeyVersionEnd       = MassifStartKeyVersionFirstByte + MassifStartKeyVersionSize
	MassifStartKeyEpochFirstByte   = MassifStartKeyVersionEnd
	MassifStartKeyEpochSize        = 4 // 32 bit
	MassifStartKeyEpochEnd         = MassifStartKeyEpochFirstByte + MassifStartKeyEpochSize
	// Note the massif height is purposefully ahead of the index, it can't be
	// changed without also incrementing the EPOCH, so we never care about it's
	// effect on the  sort order with respect to the first index
	MassifStartKeyMassifHeightFirstByte = MassifStartKeyEpochEnd
	MassifStartKeyMassifHeightSize      = 1 // 8 bit
	MassifStartKeyMassifHeightEnd       = MassifStartKeyMassifHeightFirstByte + MassifStartKeyMassifHeightSize

	MassifStartKeyMassifFirstByte     = MassifStartKeyMassifHeightEnd
	MassifStartKeyMassifSize          = 4
	MassifStartKeyMassifEnd           = MassifStartKeyMassifFirstByte + MassifStartKeyMassifSize // 32 bit
	MassifStartKeyFirstIndexFirstByte = MassifStartKeyMassifEnd

	MassifCurrentVersion = uint16(0)
)

var (
	ErrMassifFixedHeaderMissing = errors.New("the fixed header for the massif is missing")
	ErrMassifFixedHeaderBadType = errors.New("the fixed header for the massif has the wrong type code")

	ErrEntryTypeUnexpected = errors.New("the entry type was not as expected")
	ErrEntryTypeInvalid    = errors.New("the entry type was invalid")
	ErrPrevRootNotSet      = errors.New("the previous root was not provided")
)

// MassifStart defines the values encoded in the header field of every massif blob.
// The header field is written to the first 32 byte record in the blob.
// See [Massif Basic File Format](https://github.com/datatrails/epic-8120-scalable-proof-mechanisms/blob/main/mmr/forestrie-mmrblobs.md#massif-basic-file-format)
type MassifStart struct {
	MassifHeight    uint8
	DataEpoch       uint8
	Version         uint16
	CommitmentEpoch uint32
	MassifIndex     uint32
	FirstIndex      uint64
	LastID          uint64
	PeakStackLen    uint64
}

func NewMassifStart(lastID uint64, commitmentEpoch uint32, massifHeight uint8, massifIndex uint32, firstIndex uint64) MassifStart {
	return MassifStart{
		MassifHeight:    massifHeight,
		DataEpoch:       0,
		Version:         MassifCurrentVersion,
		CommitmentEpoch: commitmentEpoch,
		MassifIndex:     massifIndex,
		FirstIndex:      firstIndex,
		LastID:          lastID,
	}
}

// MassifFirstLeaf returns the MMR index of the first leaf in the massif blob identified by massifIndex
func MassifFirstLeaf(massifHeight uint8, massifIndex uint32) uint64 {

	// The number of leaves 'f' in a massif is derived from its height.

	// Given massif height, the number of m nodes is:
	// 	m = (1 << h) - 1
	m := uint64((1 << massifHeight) - 1)

	// The size can be computed from the number of leaves f as
	// 	m = f + f - 1
	//
	// So to recover the number of f leaves in every massif in the epoch from m we have:
	// 	f = (m + 1) / 2
	f := (m + 1) / 2

	// So the first *leaf* index is then
	leafIndex := f * uint64(massifIndex)

	// And now we can apply TreeIndex to the leaf index. This last is an
	// iterative call but it is sub linear. Essentially its O(tree height) (not
	// massif height ofc)
	return mmr.TreeIndex(leafIndex)
}

func (ms MassifStart) MarshalBinary() ([]byte, error) {
	return EncodeMassifStart(ms.LastID, ms.Version, ms.CommitmentEpoch, ms.MassifHeight, ms.MassifIndex), nil
}

func (ms *MassifStart) UnmarshalBinary(b []byte) error {
	return DecodeMassifStart(ms, b)
}

// EncodeMassifStart encodes the massif details in the prescribed massif header
// record format
//
// .         | <reserved>|lastid |<reserved>|   version| epoch  |massif height| massif i |
// .         |           | 8-16  |          |  21 - 22 | 23   26|27         27| 28 -  31 |
// bytes     |           |       |          |      2   |    4   |      1      |     4    |
func EncodeMassifStart(lastID uint64, version uint16, epoch uint32, massifHeight uint8, massifIndex uint32) []byte {

	start := make([]byte, StartHeaderSize)

	binary.BigEndian.PutUint64(start[MassifStartKeyLastIDFirstByte:MassifStartKeyLastIDEnd], lastID)
	binary.BigEndian.PutUint16(start[MassifStartKeyVersionFirstByte:MassifStartKeyVersionEnd], version)
	binary.BigEndian.PutUint32(start[MassifStartKeyEpochFirstByte:MassifStartKeyEpochEnd], epoch)
	start[MassifStartKeyMassifHeightFirstByte] = massifHeight
	binary.BigEndian.PutUint32(start[MassifStartKeyMassifFirstByte:MassifStartKeyMassifEnd], massifIndex)
	return start
}

func DecodeMassifStart(ms *MassifStart, start []byte) error {
	if len(start) < (ValueBytes) {
		return ErrMassifFixedHeaderBadType
	}

	ms.LastID = binary.BigEndian.Uint64(start[MassifStartKeyLastIDFirstByte:MassifStartKeyLastIDEnd])
	ms.Version = binary.BigEndian.Uint16(start[MassifStartKeyVersionFirstByte:MassifStartKeyVersionEnd])
	ms.CommitmentEpoch = binary.BigEndian.Uint32(start[MassifStartKeyEpochFirstByte:MassifStartKeyEpochEnd])
	ms.MassifHeight = start[MassifStartKeyMassifHeightFirstByte]

	ms.MassifIndex = binary.BigEndian.Uint32(start[MassifStartKeyMassifFirstByte:MassifStartKeyMassifEnd])
	ms.FirstIndex = MassifFirstLeaf(ms.MassifHeight, ms.MassifIndex)
	ms.PeakStackLen = mmr.LeafMinusSpurSum(uint64(ms.MassifIndex))

	return nil
}
