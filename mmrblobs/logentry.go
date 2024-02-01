package mmrblobs

import "errors"

// To enable exclusion proofs and history independent proof of completion we
// assemble the log as an array of KEY, VALUE. Each is both 32 bytes.

const (
	ValueBits             = 128
	ValueBytes            = 32
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

type LogEntry struct {
	Data []byte
}

// IndexedValue returns the value bytes from log data corresponding to entry
// index i. No range checks are performed, out of range will panic
func IndexedLogValue(logData []byte, i uint64) []byte {
	return logData[i*LogEntryBytes : i*LogEntryBytes+ValueBytes]
}

func (le LogEntry) Value() []byte {
	return le.Data[32:64]
}

func (le LogEntry) Entry() []byte {
	return le.Data
}

func (le *LogEntry) CopyBytes(b []byte) int {
	le.Data = make([]byte, ValueBytes)
	return copy(le.Data, b)
}

func (le *LogEntry) SetBytes(b []byte) error {
	if len(b) < (1 << EntryByteSizeLogBase2) {
		return ErrLogEntryToSmall
	}
	le.Data = b
	return nil
}
