package mmr

import "hash"

// HashPosPair64 returns H(pos || a || b)
// ** the hasher is reset **
func HashPosPair64(hasher hash.Hash, pos uint64, a []byte, b []byte) []byte {
	hasher.Reset()
	HashWriteUint64(hasher, pos)
	hasher.Write(a)
	hasher.Write(b)
	return hasher.Sum(nil)
}
