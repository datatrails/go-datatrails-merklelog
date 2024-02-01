package mmrblobs

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
// information is recovered computationaly computed based on the blobs possition
// in the MMR
//
// The massif start field is also trie key compatible so that the start data can
// be included in the history indpendent proofs of exclusion and of
// completeness.

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/datatrails/go-datatrails-merklelog/mmr"
)

type KeyType uint8

const (
	KeyTypeApplicationContent KeyType = iota // this is the standard entry type, purposefuly defined as 0
	_
	_
	_
	_
	_
	_
	_
	KeyTypeApplicationLast // first 8 types are reserved for the application
	// other entries are reserved for the MMR book keeping

	// KeyTypeInteriorNode trie keys for MMR interior nodes have this type
	KeyTypeInteriorNode

	// KeyTypeMassifStart is the type for keys which correspond to massif blob
	// header values
	KeyTypeMassifStart
	KeyTypeMax
)

const (

	// MassifStart layout
	//
	// .         | type| <reserved>|   version| epoch  |massif height| massif i |
	// .         | 0   |           |  21 - 22 | 23   26|27         27| 28 -  31 |
	// bytes     | 1   |           |      2   |    4   |      1      |     4    |
	//
	// Note this layout produces a sequentially valued key. The value is always
	// considered as a big endian large integer. Lexical ordering is defined
	// only for padded hex representations of the key value. The reserved zero
	// bytes can be used in later versions.  Because if we shift the version
	// field left, even without incrementing it, the resulting key is
	// numerically larger than all of those for previous versions

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
	ErrMassifBelowMinSize  = errors.New("a massive blob always has at least three log entries")
	ErrPrevRootNotSet      = errors.New("the previous root was not provided")
)

type MassifStart struct {
	MassifHeight uint8
	Version      uint16
	Epoch        uint32
	MassifIndex  uint32
	FirstIndex   uint64
	PeakStackLen uint64
}

func NewMassifStart(epoch uint32, massifHeight uint8, massifIndex uint32, firstIndex uint64) MassifStart {
	return MassifStart{
		Version:      MassifCurrentVersion,
		MassifHeight: massifHeight,
		MassifIndex:  massifIndex,
		FirstIndex:   firstIndex,
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
	return EncodeMassifStart(ms.Version, ms.Epoch, ms.MassifHeight, ms.MassifIndex), nil
}

func (ms *MassifStart) UnmarshalBinary(b []byte) error {
	return DecodeMassifStart(ms, b)
}

// EncodeMassifStart encodes the massif details in the prescribed massif header
// record format
//
// .         | type| <reserved>|   version| epoch  |massif height| massif i |
// .         | 0   |           |  21 - 22 | 23   26|27         27| 28 -  31 |
// bytes     | 1   |           |      2   |    4   |      1      |     4    |
func EncodeMassifStart(version uint16, epoch uint32, massifHeight uint8, massifIndex uint32) []byte {
	key := [32]byte{}

	key[0] = byte(KeyTypeMassifStart)

	binary.BigEndian.PutUint16(key[MassifStartKeyVersionFirstByte:MassifStartKeyVersionEnd], version)
	binary.BigEndian.PutUint32(key[MassifStartKeyEpochFirstByte:MassifStartKeyEpochEnd], epoch)
	key[MassifStartKeyMassifHeightFirstByte] = massifHeight
	binary.BigEndian.PutUint32(key[MassifStartKeyMassifFirstByte:MassifStartKeyMassifEnd], massifIndex)
	return key[:]
}

func DecodeMassifStart(ms *MassifStart, start []byte) error {
	if len(start) < (ValueBytes) {
		return ErrMassifFixedHeaderBadType
	}

	if KeyType(start[0]) != KeyTypeMassifStart {
		return fmt.Errorf("%w: %d", ErrMassifFixedHeaderBadType, start[0])
	}

	ms.Version = binary.BigEndian.Uint16(start[MassifStartKeyVersionFirstByte:MassifStartKeyVersionEnd])
	ms.Epoch = binary.BigEndian.Uint32(start[MassifStartKeyEpochFirstByte:MassifStartKeyEpochEnd])
	ms.MassifHeight = start[MassifStartKeyMassifHeightFirstByte]

	ms.MassifIndex = binary.BigEndian.Uint32(start[MassifStartKeyMassifFirstByte:MassifStartKeyMassifEnd])
	ms.FirstIndex = MassifFirstLeaf(ms.MassifHeight, ms.MassifIndex)
	ms.PeakStackLen = mmr.LeafMinusSpurSum(uint64(ms.MassifIndex))

	return nil
}
