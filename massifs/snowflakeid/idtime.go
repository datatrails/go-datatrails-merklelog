package snowflakeid

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

var (
	ErrMilliEpochOverflow = errors.New("our epoch allows for up to 2^40 milliseconds")
)

func IDUnixMilli(id uint64, epoch uint8) (int64, error) {
	ms, _ := IDMilliSplit(id)
	startMS := uint64(EpochMS(epoch))
	if ms+startMS > math.MaxInt64 {
		return 0, fmt.Errorf("%d to large (when added to epoch start): %w", ms, ErrMilliEpochOverflow)
	}
	return int64(startMS + ms), nil
}

// IDMilliSplit splits the milliseconds from the sequence uniqueness data without loss
//
// Returns
//
//	milliseconds since epoch
//	the machine id and sequence, guaranteed to be < 2^24
func IDMilliSplit(id uint64) (uint64, uint32) {

	ms := id >> TimeShift

	machineSeq := make([]byte, 4)
	binary.BigEndian.PutUint32(machineSeq, uint32(id&(^TimeMask)))

	// Check its in the correct range
	machineSeqNumber := binary.BigEndian.Uint32(machineSeq)

	return ms, machineSeqNumber
}
