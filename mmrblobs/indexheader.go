package mmrblobs

// IndexHeader exists as a place holder for index specific header information.
// It reserves a single 32 byte slot and puts nothing in it.
type IndexHeader struct {
	Index uint64
}
