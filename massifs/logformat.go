package massifs

import (
	"errors"

	"github.com/datatrails/forestrie/go-forestrie/mmr"
)

const (
	// These constants are used to derive the size of the mmrblob format sections described at
	// https://github.com/datatrails/epic-8120-scalable-proof-mechanisms/blob/main/mmr/forestrie-mmrblobs.md#massif-basic-file-format

	// ValueBytes defines the width of ALL entries in the log. This fixed width
	// makes it possible to compute mmr current sizes based on knowing only the
	// massif height and the number of bytes in the file.
	ValueBytes = 32
	// ReservedHeaderSlots reserves a place to put the urkle trie root, used for
	// data recovery and proofs of exclusion, and any related material. And it
	// gives us a little flex in the data format for the initial launch of
	// forestrie. It would be frustrating to need a data migration for want of a
	// few bytes.
	ReservedHeaderSlots   = 7 // reserves n * ValueBytes at the front of the blob
	StartHeaderSize       = ValueBytes + ValueBytes*ReservedHeaderSlots
	StartHeaderEnd        = StartHeaderSize
	IndexHeaderBytes      = 32
	LogEntryBytes         = 32
	EntryByteSizeLogBase2 = 5
	ValueBitSizeLogBase2  = 8
	ValueByteSizeLogBase2 = 5
)

var (
	ErrLogEntryToSmall = errors.New("to few bytes to represent a valid log entry")
	ErrLogValueToSmall = errors.New("to few bytes to represent a valid log value")
	ErrLogValueBadSize = errors.New("log value size invalid")
)

func IndexFromBlobSize(size int) uint64 {
	if size == 0 {
		return 0
	}
	return uint64(size>>EntryByteSizeLogBase2) - 1
}

// IndexedValue returns the value bytes from log data corresponding to entry
// index i. No range checks are performed, out of range will panic.  This
// function assumes log data is sliced to the appropriate section for i to make
// sense (be it a leaf index or an mmrIndex)
func IndexedLogValue(logData []byte, i uint64) []byte {
	return logData[i*LogEntryBytes : i*LogEntryBytes+ValueBytes]
}

// FixedHeaderEnd returns the index of the first byte after the fixed header
func FixedHeaderEnd() uint64 {
	return ValueBytes + ReservedHeaderSlots*ValueBytes
}

// TrieHeaderStart returns the first byte of the index header data
// (currently this is empty)
func TrieHeaderStart() uint64 {
	return FixedHeaderEnd()
}

// TrieHeaderEnd returns the end of the bytes reserved for the trie header data.
// Currently, nothing is stored in this.
func TrieHeaderEnd() uint64 {
	return TrieHeaderStart() + IndexHeaderBytes
}

// TrieDataEntryCount returns the number of items in the Trie data
func TrieDataEntryCount(massifHeight uint8) uint64 {

	// massifHeight is a zero based index

	// see [mmr-math-cheatsheet.md](../../mmr-math-cheatsheet.md) for derivation
	// of leaf count from height
	return 1 << massifHeight
}

// TrieDataSize returns the number of bytes pre allocated to the trie entry data.
func TrieDataSize(massifHeight uint8) uint64 {
	count := TrieDataEntryCount(massifHeight)
	return count * TrieEntryBytes
}

// TrieDataStart returns the first byte of the trie entry data.
func TrieDataStart() uint64 {
	return TrieHeaderEnd()
}

// TrieDataEnd returns the first byte after the trie entry data.
func TrieDataEnd(massifHeight uint8) uint64 {
	start := TrieDataStart()
	// XXX: TODO: this allocates double what we need. massifHeight is one based and we should do
	// return start + TrieEntryBytes*(1<<(massifHeight-1))
	return start + TrieEntryBytes*(1<<massifHeight)
}

// PeakStackStart returns the first byte of the massif ancestor peak stack data
func PeakStackStart(massifHeight uint8) uint64 {
	return TrieDataEnd(massifHeight)
}

// PeakStackLen returns the number of items in the ancestor peak stack
func PeakStackLen(massifIndex uint64) uint64 {
	return mmr.LeafMinusSpurSum(massifIndex)
}

// PeakStackEnd returns the first byte after the massif ancestor peak stack data
func PeakStackEnd(massifIndex uint64, massifHeight uint8) uint64 {
	stackLen := PeakStackLen(massifIndex)
	start := PeakStackStart(massifHeight)
	return start + stackLen*ValueBytes
}

// MassifLogEntries calculates the number of log entries (nodes) in a blob from
// the length of the blob in bytes. It does this by accounting for the trie
// entries and other header data. If you know the FirstIndex from the massif
// start header you can get the overall massif size by direct addition.
//
// Note: this function exists so we can compute the mmrSize from just the azure
// blob store metadata: we store the FirstIndex on a blob tag, and the blob
// metadata includes ContentLength. This means when we are checking if a root
// seal covers the current log head, we don't need to fetch the log massif blob.
func MassifLogEntries(dataLen int, massifIndex uint64, massifHeight uint8) (uint64, error) {
	stackEnd := PeakStackEnd(massifIndex, massifHeight)
	if uint64(dataLen) < stackEnd {
		return 0, ErrMassifDataLengthInvalid
	}
	mmrByteCount := uint64(dataLen) - stackEnd
	return mmrByteCount / ValueBytes, nil
}
