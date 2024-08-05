package mmr

import "math/bits"

func BitLength64(num uint64) uint64 { return uint64(BitLength(num)) }
func BitLength(num uint64) int {
	return bits.Len64(num)
}

// Log2Uint64 efficiently computes log base 2 of num
func Log2Uint64(num uint64) uint64 {
	return uint64(bits.Len64(num) - 1)
}

func Log2Uint32(num uint32) uint32 {
	return uint32(bits.Len32(num) - 1)
}

func AllOnes(num uint64) bool {
	return (1<<bits.OnesCount64(num) - 1) == num
}
