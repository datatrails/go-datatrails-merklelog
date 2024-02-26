package mmr

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testDb struct {
	t     *testing.T
	store map[uint64][]byte
	next  uint64
}

func hashWriteUint64(hasher hash.Hash, value uint64) {
	b := [8]byte{}
	binary.BigEndian.PutUint64(b[:], value)
	hasher.Write(b[:])
}

func NewTestDb(t *testing.T) *testDb {
	db := testDb{
		t: t, store: make(map[uint64][]byte),
		next: uint64(0),
	}
	return &db
}

func NewGeneratedTestDB(t *testing.T, mmrSize uint64) *testDb {
	db := &testDb{
		t: t, store: make(map[uint64][]byte),
	}
	leafCount := LeafCount(mmrSize)
	for i := uint64(0); i < leafCount; i++ {
		_, err := AddHashedLeaf(db, sha256.New(), hashNum(TreeIndex(i)))
		require.NoError(t, err)
	}
	return db
}

// TestNode30ProofInSize63 this tests a very special case in the consistency
// proofs, which caused a lot of confusion, works in a raw mmr (rather than the
// blobbed version)
func TestNode30ProofInSize63(t *testing.T) {

	db := NewGeneratedTestDB(t, 63)
	n30 := db.mustGet(30)
	n61 := db.mustGet(61) // the only element in the proof
	root := db.mustGet(62)

	h := sha256.New()
	hashWriteUint64(h, 63)
	h.Write(n30)
	h.Write(n61)
	result := h.Sum(nil)
	ok := bytes.Equal(result, root)
	assert.True(t, ok)
}

// TestGeneratedTestDB tests that the generated db matches the canonical db if
// we generate the same size.
func TestGeneratedTestDB(t *testing.T) {
	canon := NewCanonicalTestDB(t)
	mmrSize := canon.Next()
	db := NewGeneratedTestDB(t, mmrSize)

	ok := uint64(0)
	for i := uint64(0); i < mmrSize; i++ {
		if bytes.Compare(canon.mustGet(i), db.mustGet(i)) != 0 {
			fmt.Printf("%d %d\n", i, LeafCount(i))
			continue
		}
		ok++
	}
	assert.Equal(t, ok, mmrSize)
}

// NewCanonicalTestDB populates a test data base with mmr size = 39 and where
// the leaf hashes are the hashes of the leaf positions. This tree is
// constructed mandraulicly so is suitable for tests which cover the tree
// building itself.
//
// Note that any mmr size < 39 is also contained in this MMR. So tests that want
// to work with smaller trees can just use this one but pretend its only however
// big they need.
func NewCanonicalTestDB(t *testing.T) *testDb {

	// the 1 based tree
	//	3            15
	//	           /    \
	//	          /      \
	//	         /        \
	//	2       7          14             22
	//	      /   \       /   \          /   \
	//	1    3     6    10     13      18      21      25
	//	    / \  /  \   / \   /  \    /  \    /  \    /   \
	//	0  1   2 4   5 8   9 11   12 16   17 19   20 23   24   26

	// the 0 based tree
	// 3              14
	//              /    \
	//             /      \
	//            /        \
	//           /          \
	// 2        6            13           21
	//        /   \        /    \
	// 1     2     5      9     12     17     20     24
	//      / \   / \    / \   /  \   /  \
	// 0   0   1 3   4  7   8 10  11 15  16 18  19 22  23   25

	// 4                         30
	//
	//
	// 3              14                       29
	//              /    \
	//           /          \
	// 2        6            13           21             28                37
	//        /   \        /    \
	// 1     2     5      9     12     17     20     24       27       33      36
	//      / \   / \    / \   /  \   /  \
	// 0   0   1 3   4  7   8 10  11 15  16 18  19 22  23   25   26  31  32   34  35   38
	// .   0 . 1 2 . 3 .4 . 5  6 . 7  8 . 9 10  11 12  13   14   15  16  17   18  19   20

	// XXX: TODO update this for position commitment in interior nodes
	db := testDb{
		t: t, store: make(map[uint64][]byte),
		// next: uint64(19),
		next: uint64(39),
	}

	// height 0 (the leaves)
	db.put(0, hashNum(0))
	db.put(1, hashNum(1))
	db.put(3, hashNum(3))
	db.put(4, hashNum(4))
	db.put(7, hashNum(7))
	db.put(8, hashNum(8))
	db.put(10, hashNum(10))
	db.put(11, hashNum(11))
	db.put(15, hashNum(15))
	db.put(16, hashNum(16))
	db.put(18, hashNum(18))
	db.put(19, hashNum(19))
	db.put(22, hashNum(22))
	db.put(23, hashNum(23))
	db.put(25, hashNum(25))
	db.put(26, hashNum(26))
	db.put(31, hashNum(31))
	db.put(32, hashNum(32))
	db.put(34, hashNum(34))
	db.put(35, hashNum(35))
	db.put(38, hashNum(38))

	// height 1
	db.put(2, db.hashPair(2+1, 0, 1))
	db.put(5, db.hashPair(5+1, 3, 4))
	db.put(9, db.hashPair(9+1, 7, 8))
	db.put(12, db.hashPair(12+1, 10, 11))
	db.put(17, db.hashPair(17+1, 15, 16))
	db.put(20, db.hashPair(20+1, 18, 19))
	db.put(24, db.hashPair(24+1, 22, 23))
	db.put(27, db.hashPair(27+1, 25, 26))
	db.put(33, db.hashPair(33+1, 31, 32))
	db.put(36, db.hashPair(36+1, 34, 35))

	// height 2
	db.put(6, db.hashPair(6+1, 2, 5))
	db.put(13, db.hashPair(13+1, 9, 12))
	db.put(21, db.hashPair(21+1, 17, 20))
	db.put(28, db.hashPair(28+1, 24, 27))
	db.put(37, db.hashPair(37+1, 33, 36))

	// height 3
	db.put(14, db.hashPair(14+1, 6, 13))
	db.put(29, db.hashPair(29+1, 21, 28))

	// height 4
	db.put(30, db.hashPair(30+1, 14, 29))

	return &db
}

func (db *testDb) Next() uint64 {
	return db.next
}

func (db *testDb) Append(value []byte) (uint64, error) {
	db.store[db.next] = value
	db.next += 1
	return db.next, nil
}

func (db *testDb) Get(i uint64) ([]byte, error) {
	if value, ok := db.store[i]; ok {
		return value, nil
	}
	return nil, ErrNotFound
}

func (db *testDb) mustGet(i uint64) []byte {
	if value, err := db.Get(i); err == nil {
		return value
	}
	db.t.Fatalf("index %v not found", i)
	return nil
}

// Put is provided for testing purposes only, the mmr does not use Put at all
func (db *testDb) put(i uint64, value []byte) {
	if _, ok := db.store[i]; ok {
		db.t.Fatalf("index %v already set", i)
	}
	db.store[i] = value
	if db.next < i {
		db.next = i + 1
	}
}

func (db *testDb) hashPair(pos, i, j uint64) []byte {

	var err error
	var value []byte

	h := sha256.New()
	bytes8 := make([]byte, 8)

	binary.BigEndian.PutUint64(bytes8, pos) // commit to the count (pos) rather than index
	h.Write(bytes8)

	if value, err = db.Get(i); err != nil {
		db.t.Fatalf("index %v not found", i)
	}
	// XXX: TODO: position commitment for inner leaves
	h.Write(value)
	if value, err = db.Get(j); err != nil {
		db.t.Fatalf("index %v not found", i)
	}
	h.Write(value)

	return h.Sum(nil)
}

func hashNum(num uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, num)
	h := sha256.New()
	h.Write(b)
	return h.Sum(nil)
}
