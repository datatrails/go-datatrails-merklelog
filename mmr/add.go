package mmr

import (
	"hash"
)

type NodeAppender interface {
	Get(i uint64) ([]byte, error)
	Append(value []byte) (uint64, error)
}

// AddHashedLeaf adds a single leaf to the mmr and back fills any interior nodes
// 'above and to the left'
//
// Returns the size of the mmr after addition of the leaf. This is also the
// position of the next leaf.
func AddHashedLeaf(store NodeAppender, hasher hash.Hash, hashedLeaf []byte) (uint64, error) {

	var err error
	var i uint64

	hasher.Reset()
	height := uint64(0) // leaf height is always zero

	if i, err = store.Append(hashedLeaf); err != nil {
		return 0, err
	}

	// This loop checks to see if we can back fill any new mountains. Because of
	// the MMR structure, for any node we add, if the next node after that would
	// be higher in the tree, then the node we just added lets us create at
	// least one new peak.
	//
	// Here, we add the second item, and it lets us add the first peak at 2
	//
	//  0 1 <- we add '1'
	//
	//   2  <- so we get to append '2' as well, because the iNext would be higher
	//  / \
	// 0   1
	//
	// This tactic works no matter how many peaks exist already, as each
	// backfilled 'peak' is always at the 'next' position relative to the node
	// that was just added. Hence we get away with this deceptively simple
	// looking iterative approach.
	//
	// Note that i is at 'next' every time we call IndexHeight
	for IndexHeight(i) > height {

		iLeft := i - (2 << height)
		// iRight is always just i - 1
		// because i - (2 << height ) + SiblingOffset(height)
		// 		=> i - (2 << height ) + (2 << height) - 1
		// 		=> i - 1
		iRight := i - 1

		hasher.Reset()

		// For interior nodes, commit to the position to provide for non
		// equivocal proof of position see: ref:
		// https://github.com/proofchains/python-proofmarshal/blob/master/proofmarshal/mmr.py#L142
		HashWriteUint64(hasher, i+1)

		var value []byte

		if value, err = store.Get(iLeft); err != nil {
			return 0, err
		}
		hasher.Write(value)

		if value, err = store.Get(iRight); err != nil {
			return 0, err
		}
		hasher.Write(value)

		if i, err = store.Append(hasher.Sum(nil)); err != nil {
			return 0, err
		}
		height += 1
	}
	return i, nil
}
