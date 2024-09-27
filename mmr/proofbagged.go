package mmr

import (
	"hash"
	"slices"
)

// Verification is by default against the MRR accumulator peaks (see verify.go). The "Bagged"
// variants work with proofs against a "Bagged" singular mono root for an MMR.

// GetRoot returns the root hash for the Merkle Mountain Range.
// The root is defined as the 'bagging' of all peaks, starting with the highest.
// So its simply a call to BagPeaksRHS for _all_ peaks in the MMR of the provided size.
func GetRoot(mmrSize uint64, store indexStoreGetter, hasher hash.Hash) ([]byte, error) {
	peaks := PosPeaks(mmrSize)
	// The root is ALL the peaks. Note that bagging essentially accumulates them in a binary tree.
	return BagPeaksRHS(store, hasher, 0, peaks)
}

// IndexProofBagged provides a proof of inclusion for the leaf at index i against the full MMR
//
// It relies on the methods IndexProofLocal, BagPeaksRHS and PeaksLHS for
// collecting the necessary MMR elements and then combines the results into a
// final verifiable commitment for the whole MMR.
//
// The proof layout is conceptualy this:
//
// [local-peak-proof-i, right-sibling-of-i, left-of-i-peaks-reversed]
//
// So for leaf 15, given
//
//	3              14
//	             /    \
//	            /      \
//	           /        \
//	          /          \
//	2        6            13           21
//	       /   \        /    \
//	1     2     5      9     12     17     20     24
//	     / \   / \    / \   /  \   /  \
//	0   0   1 3   4  7   8 10  11 15  16 18  19 22  23   25
//
// We get
//
//	               |  BagPeaksRHS   |
//	               .                .
//	[H(16), H(20), H(H(H(25)|H(24)) | H(21), H(14)]
//	 ^ .         ^                   ^           ^
//	 .___________.                   .___________.
//	       |                               |
//	       |                          reversed(PeaksLHS)
//	   IndexProofPath
//
// Note that right-sibling is omitted if there is none, and similarly, the left
// peaks. The individual functions producing those elements contain more detail
// over the construction of their particular proof component.
func IndexProofBagged(mmrSize uint64, store indexStoreGetter, hasher hash.Hash, i uint64) ([][]byte, error) {

	var err error
	var proof [][]byte
	var iLocalPeak uint64 // the peak of the local merkle tree containing i
	var leftPath [][]byte
	var rightSibling []byte

	if proof, iLocalPeak, err = IndexProofLocal(mmrSize, store, i); err != nil {
		return nil, err
	}

	peaks := PosPeaks(mmrSize)

	if rightSibling, err = BagPeaksRHS(store, hasher, iLocalPeak+1, peaks); err != nil {
		return nil, err
	}
	if rightSibling != nil {
		proof = append(proof, rightSibling)
	}

	if leftPath, err = PeaksLHS(store, iLocalPeak+1, peaks); err != nil {
		return nil, err
	}
	// reverse(leftPath)
	slices.Reverse(leftPath)
	proof = append(proof, leftPath...)

	return proof, nil
}

// BagPeaksRHS computes a root for the RHS peaks.
// This function will only return an err if there is an issue fetching a value
// from the provided store.
//
// The burden is on the _caller_ to provide valid peaks for the given position pos
//
// If there are no peaks to the right of pos, this function returns nil, nil. This
// means the sibling hash for pos is to the left and the return value should be
// ignored.
//
// Working exclusively in positions rather than indices, If the peak pos is 25,
// then the RHS (and the sibling hash) is just H(26), if pos is 26 then there is
// not right sibling, and this method would return nil.
//
// The peaks are listed in ascending order (ie from the end of
// the range back towards pos), So when pos is 15, the RHS sibling hash will
// be:
//
//	H(H(H(right)|H(left)) | H(22))
//
// Which is:
//
//	H(H(H(26)|H(25)) | H(22))
//
//	3            15
//	           /    \
//	          /      \
//	         /        \
//	2       7          14             22
//	      /   \       /   \          /   \
//	1    3     6    10     13      18      21      25
//	    / \  /  \   / \   /  \    /  \    /  \    /   \
//	0  1   2 4   5 8   9 11   12 16   17 19   20 23   24 26
func BagPeaksRHS(store indexStoreGetter, hasher hash.Hash, pos uint64, peaks []uint64) ([]byte, error) {

	peakHashes, err := PeakBagRHS(store, hasher, pos, peaks)
	if err != nil {
		return nil, err
	}

	root := hashPeaksRHS(hasher, peakHashes)
	return root, nil
}

// PeakBagRHS collects the peaks for BagPeaksRHS in the right order for hashing
func PeakBagRHS(
	store indexStoreGetter, hasher hash.Hash, pos uint64, peaks []uint64) ([][]byte, error) {
	var err error
	var value []byte
	var peakHashes [][]byte

	for _, peakPos := range peaks {

		// skip all left peaks and the pos peak
		if peakPos <= pos {
			continue
		}
		// As the leaves are indexed from zero, we just do pos - 1 to access the leaf.
		if value, err = store.Get(peakPos - 1); err != nil {
			return nil, err
		}
		peakHashes = append(peakHashes, value)
	}
	return peakHashes, nil
}

// IndexProofLocal collects the merkle root proof for the local MMR peak containing index i
//
// So for the follwing index tree, and i=15 with mmrSize = 26 we would obtain the path
//
// [H(16), H(20)]
//
// Because the local peak is 21, and given the value for 15, we only need 16 and
// then 20 to prove the local root.
//
//	3              14
//	             /    \
//	            /      \
//	           /        \
//	          /          \
//	2        6            13           21
//	       /   \        /    \
//	1     2     5      9     12     17     20     24
//	     / \   / \    / \   /  \   /  \
//	0   0   1 3   4  7   8 10  11 15  16 18  19 22  23   25
func IndexProofLocal(mmrSize uint64, store indexStoreGetter, i uint64) ([][]byte, uint64, error) {

	var proof [][]byte
	height := IndexHeight(i) // allows for proofs of interior nodes

	var err error
	var value []byte

	for i < mmrSize {
		iHeight := IndexHeight(i)
		iNextHeight := IndexHeight(i + 1)
		if iNextHeight > iHeight {
			iSibling := i - SiblingOffset(height)
			if iSibling >= mmrSize {
				break
			}

			if value, err = store.Get(iSibling); err != nil {
				return nil, 0, err
			}
			proof = append(proof, value)
			// go to parent node
			i += 1
		} else {
			iSibling := i + SiblingOffset(height)
			if iSibling >= mmrSize {
				break
			}

			if value, err = store.Get(iSibling); err != nil {
				return nil, 0, err
			}
			proof = append(proof, value)
			// goto parent node
			i += 2 << height
		}
		height += 1
	}
	return proof, i, nil
}

// hashPeaksRHS creates a binary merkle tree from the peaks to obtain a single
// tree root.
//
// WARNING: MUTATES the input slice by popping items from it
func hashPeaksRHS(hasher hash.Hash, peakHashes [][]byte) []byte {

	var right []byte
	var left []byte

	// The hashes are highest to lowest, we are popping so we consume from the end backwards.
	for len(peakHashes) > 1 {

		right, peakHashes = peakHashes[len(peakHashes)-1], peakHashes[:len(peakHashes)-1] // go lang's array pop
		left, peakHashes = peakHashes[len(peakHashes)-1], peakHashes[:len(peakHashes)-1]  // go lang's array pop

		hasher.Reset()
		hasher.Write(right)
		hasher.Write(left)

		peakHashes = append(peakHashes, hasher.Sum(nil))
	}
	if len(peakHashes) > 0 {
		return peakHashes[0]
	}
	return nil
}

// HashPeaksRHS merkleizes the peaks to obtain a single tree root
// This variant copies the peakHashes list in order to be side effect free.
func HashPeaksRHS(hasher hash.Hash, peakHashes [][]byte) []byte {
	return hashPeaksRHS(hasher, append([][]byte(nil), peakHashes...))
}

// PeaksLHS collects the peaks to the left of position pos into a flat sequence
//
// So for the following tree and pos=25 we would get
//
//	[15, 22]
//
//	3            15
//	           /    \
//	          /      \
//	         /        \
//	2       7          14             22
//	      /   \       /   \          /   \
//	1    3     6    10     13      18      21      25
//	    / \  /  \   / \   /  \    /  \    /  \    /   \
//	0  1   2 4   5 8   9 11   12 16   17 19   20 23   24 26
func PeaksLHS(store indexStoreGetter, pos uint64, peaks []uint64) ([][]byte, error) {

	var err error
	var value []byte
	var peakHashes [][]byte

	for _, peakPos := range peaks {
		if peakPos >= pos {
			break
		}
		if value, err = store.Get(peakPos - 1); err != nil {
			return nil, err
		}
		peakHashes = append(peakHashes, value)
	}
	return peakHashes, nil
}
