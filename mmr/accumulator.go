package mmr

type AccumulatorMask uint64

// AccumulatorSparse records the accumulator peak positions. zero indicates unoccupied
type AccumulatorSparse []uint64

// AccumulatorIndices records the packed peak indices. The highest peak is first.
type AccumulatorIndices []uint64
