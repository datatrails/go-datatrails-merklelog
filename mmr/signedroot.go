package mmr

type SignedMerkleRoot struct {
	// Timestamp is the unix time read at the time the root was signed
	Timestamp uint64 `cbor:"1,keyasint"`
	MMRSize   uint64 `cbor:"2,keyasint"`
	Root      []byte `cbor:"3,keyasint"`
}
