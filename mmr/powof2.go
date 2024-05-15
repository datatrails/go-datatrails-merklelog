package mmr

// IsPow2 determins if the unsigned value size is a perfect power of 2.
func IsPow2(size uint) bool {
	if size == 0 {
		return false
	}
	// shift down (in size) until the first set bit is found
	for ; (size & 1) == 0; size >>= 1 {
	}

	// If any other bits are set, then its not a perfect power of 2. Remember ^
	// is go langs only way to do logical compliment.
	return (size & ^uint(1)) == 0
}
