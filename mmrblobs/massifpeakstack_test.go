//go:build integration && azurite

package mmrblobs

import (
	"fmt"
	"testing"

	"github.com/datatrails/forestrie/triecommon/testgenerator"
	"github.com/datatrails/go-datatrails-merklelog/mmr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPeakStack_popArithmetic tests that the primitive methods the massif peak stack
// relies on and the arithmetic for maintaining the stack are consistent.
func TestPeakStack_popArithmetic(t *testing.T) {

	// Working with height 2 massifs and the following overall MMR
	//
	//  4                        30
	//
	//
	//               14                        29
	//	3           /  \                      /   \
	//	           /    \                    /     \
	//	          /      \                  /       \
	//	         /        \                /         \
	//	2      6 .      .  13             21          28
	//	      /   \       /   \          /  \        /   \
	//	1    2  |  5  |  9  |  12   |  17  | 20   | 24   | 27   |  --- massif tree line massif height = 1
	//	    / \ |/  \ | / \ |  /  \ | /  \ | / \  | / \  | / \  |
	//	   0   1|3   4|7   8|10   11|15  16|18  19|22  23|25  26| MMR INDICES
	//     -----|-----|-----|-------|------|------|------|------|
	//	   0 . 1|2 . 3|4   5| 6    7| 8   9|10  11|12  13|14  15| LEAF INDICES
	//     -----|-----|-----|-------|------|------|------|------|
	//       0  |  1  |  2  |  3    |   4  |   5  |   6  |   7  | MASSIF INDICES
	//     -----|-----|-----|-------|------|------|------|------|

	// height, a 3 node tree has height 2 (some places we use a height index)
	massifHeight := uint64(2) // each masif has 2 leaves and 3 nodes + spur
	massifNodeCount := uint64((1 << massifHeight) - 1)
	massifLeafCount := (massifNodeCount + 1) / 2

	stack := []uint64{}

	expectStacks := [][]uint64{
		{uint64(2)},
		{uint64(6)},
		{uint64(6), uint64(9)},
		{uint64(14)},
		{uint64(14), uint64(17)},
		{uint64(14), uint64(21)},
		{uint64(14), uint64(21), uint64(24)},
		{uint64(30)},
	}

	for massifIndex := uint64(0); massifIndex < 8; massifIndex++ {

		t.Run(fmt.Sprintf("iLeaf:%d", massifIndex), func(t *testing.T) {

			// this shows that we can work with massif indices as tho they were
			// leaf indices. the only point the difference between leaf and
			// massif blob index matters is where we calculate the MMR index of
			// the node we are putitng on the stack. We never explicitly
			// calculate the index of the node being added, we just add it, its
			// the arithmetic for 'popping' the stack we care about. We track
			// the implied node indices here only for the purpose of the test.
			//
			// Note in particular, any node that gets into the stack is always
			// the *last* node from a particular massif blob. The peak nodes we
			// need to reference in future blobs are *always* last leafs from
			// some preceding blob. The MMR structure means there are 'interior'
			// peaks, but those are only referenced within that particular blob.

			lastLeaf := massifIndex*massifLeafCount + massifLeafCount - 1
			spurHeightLeaf := mmr.SpurHeightLeaf(lastLeaf)
			iPeak := mmr.TreeIndex(lastLeaf) + spurHeightLeaf

			stackLen := mmr.LeafMinusSpurSum(massifIndex)

			// we push for current leaf and pop for previous
			pop := mmr.SpurHeightLeaf(massifIndex)

			fmt.Printf("-----: L=%02d LL=%02d P=%d, StackLen=%d, pop=%d\n", massifIndex, lastLeaf, iPeak, stackLen, pop)
			fmt.Printf("stack:%v\n", stack)
			assert.Equal(t, stackLen, uint64(len(stack)))

			stack = stack[:len(stack)-int(pop)]
			// stack = append(stack, iPeak)
			stack = append(stack, iPeak)

			// Check we produced the expected stack for the next round. Note
			// that each time we start a new blob in StartNextMassif, we have
			// just read the previous and discovered that it is full. So this
			// corresponds to creating the stack for the *new* blob based on the
			// full blob we have in our hand.
			assert.Equal(t, expectStacks[massifIndex], stack)
			//fmt.Printf("i=%02d push(%d) pop-len %d: %v %v %v\n", leafIndex, iRoot, pop, stackBefore, stackPop, stack)
			// fmt.Printf("after:i=%02d r=%d: %v %v %v\n", leafIndex, iRoot, stackBefore, stackPop, stack)
		})
	}
}
func TestPeakStack_StartNextMassif(t *testing.T) {
	var err error

	// ctx := context.Background()

	tc := testgenerator.NewContext(t, testgenerator.TestConfig{
		StartTimeMS: (1698342521) * 1000, EventRate: 500,
		TestLabelPrefix: "TestPeakStack_StartNextMassif", TenantIdentity: "", Container: "StartNextMassif"})
	tenantIdentity := tc.NewTenantIdentity()
	tc.DeleteTenantMassifs(tenantIdentity)

	massifHeight := uint8(2) // each masif has 2 leaves and 3 nodes + spur
	mc := MassifContext{TenantIdentity: tenantIdentity, Tags: make(map[string]string)}
	mc.Start = NewMassifStart(0, massifHeight, 0, 0)
	mc.Data, err = mc.Start.MarshalBinary()
	mc.Data = append(mc.Data, mc.initIndexData()...)
	require.Nil(t, err)

	// The following two helpers assist checking consistency between the
	// ancestor peak stack and the log
	getFromData := func(mc MassifContext, i uint64) []byte {

		logStart := mc.LogStart()
		start := logStart + i*ValueBytes
		end := start + ValueBytes
		if end > uint64(len(mc.Data)) {
			t.Fatalf("end of value %d at %d exceeds data size %d", i, end, len(mc.Data))
			return nil
		}
		return mc.Data[start:end]
	}
	getFromStack := func(mc MassifContext, i uint64) []byte {
		if i > mc.Start.PeakStackLen {
			t.Fatalf("%d exceeds stack len %d", i, mc.Start.PeakStackLen)
			return nil
		}
		start := mc.PeakStackStart() + i*ValueBytes
		end := start + ValueBytes
		return mc.Data[start:end]
	}

	// NOTICE THis test follows the material here: https://github.com/datatrails/epic-8120-scalable-proof-mechanisms/blob/main/mmr/forestrie-mmrblobs.md#stack-maintenance
	// Some of which is reproduced in line

	// considering the following mmr
	//
	//  4                        30
	//
	//
	//               14                        29
	//	3           /  \                      /   \
	//	           /    \                    /     \
	//	          /      \                  /       \
	//	         /        \                /         \
	//	2      6 .      .  13             21          28
	//	      /   \       /   \          /  \        /   \
	//	1    2  |  5  |  9  |  12   |  17  | 20   | 24   | 27   |  --- massif tree line massif height = 1
	//	    / \ |/  \ | / \ |  /  \ | /  \ | / \  | / \  | / \  |
	//	   0   1|3   4|7   8|10   11|15  16|18  19|22  23|25  26| MMR INDICES
	//     -----|-----|-----|-------|------|------|------|------|
	//	   0 . 1|2 . 3|4   5| 6    7| 8   9|10  11|12  13|14  15| LEAF INDICES
	//     -----|-----|-----|-------|------|------|------|------|
	//       0  |  1  |  2  |  3    |   4  |   5  |   6  |   7  | MASSIF INDICES
	//     -----|-----|-----|-------|------|------|------|------|
	//
	// As the massif blobs accumulate, the peak stack maintains copies of the
	// minimal set of nodes that are required from preceding blobs in order to
	// complete the current. This set grows with log base 2 n of the *massif*
	// blob count, its never realistically going to get more than a few items
	// long. And if its size ever gets to be a problem we would just start a new
	// epoch.
	//
	// For example, when we add leaf 7 (mmr index 11), we need to use mmr
	// indices 10, 9 and 6 in order to create 11, 12, 13 and 14.
	//  The nature of addition means we will require those ancestor nodes in
	// exactly that order, and we will need them all exactly, and only, when we
	// add mmr index 11 (leaf 7), or at some arbitrary point later if we need to
	// produce a receipt for leaves 7 *or* 6. Whether we are adding mmr index 11
	// or whether we are generatint a receipt for mmr indices 6 or 7, we always
	// need ancestor mmr's 9 and 6 and in that order. The massif local nodes (10
	// or 11 in this example) are available via normal Get access directly from
	// the blob data array.
	//
	// The massif blobs are constructed from strictly 32 byte fields. Each
	// massif has a single START record which contains the mmr index occupied by
	// the first log entry in the massif, and a record of the massif height. The
	// massif height is constant through out each epoch. The current epoch is
	// also in START. See [mmrblobs.$EncodeMassifStart] and
	// [mmrblobs.$MassifStart] for precise layout. For the purpose of this test
	// only MassifIndex and FirstIndex are significant
	//
	// +----------------+
	// | START [MI, FI] | field 0, containing MassifIndex and FirstIndex, MI and FI.
	// + ---------------+
	// | PEAK STACK     | field 1 - stack len. stack len is derived via [mmr.$LeafMinusSpurSum](MassifIndex)
	// .   ...          .
	// + ---------------+
	// | First Entry    | The first log entry, which occupies MMR INDEX FirstIndex
	//
	// Layed out horizontally, the first massif will look like this
	//
	// +--------++---+---+---+
	// | [0, 0] || 0 | 1 | 2 |
	// +--------++---+-------+
	//
	// The peak stack is empty

	// --- massif 0 has exactly 3 nodes

	//	1    2  | --- massif tree line massif height = 1
	//	    / \ |
	//	   0   1| MMR INDICES
	//	   0 . 1| LEAF INDICES
	//     -----|
	//       0  | MASSIF INDICES
	// |
	// +--------++---+---+---+
	// | [0, 0] || 0 | 1 | 2 |
	// +--------++---+-------+
	//
	mc.Data = tc.PadWithNumberedLeaves(mc.Data, 0, 1<<massifHeight-1)

	var peakStack []byte

	// The ancestor stack excludes the log entry from the current massif. For massif 0 it is empty.
	peakStack, err = mc.GetAncestorPeakStack()
	assert.Nil(t, err)
	assert.Nil(t, peakStack)

	// --- massif 1

	// We begin with the data of massif 0 from above
	//
	// +--------++---+---+---+
	// | [0, 0] || 0 | 1 | 2 |
	// +--------++---+-------+
	//
	// And create the data for starting massif 1, this must include the peak stack (including the last value) from massif 0
	//
	//     stackLen(0) = 0
	//     popLen(0)   = 0
	//     pop stack   = stack[:stackLen-popLen] = stack[:0-0]
	//     push stack  = append(stack, 2) (last leaf of massif 0)
	//
	// note it is crucial we pop the items before appending the new.
	//
	// And create the data for starting massif 1, this must include the peak stack (including the last value) from massif 0
	//
	//	1    2  | --- massif tree line massif height = 1
	//	    / \ |
	//	   0   1| MMR INDICES
	//	   0 . 1| LEAF INDICES
	//       0  | MASSIF INDICES

	//	2     \ 6
	//	      /\  \
	//	1    2  |  5  | --- massif tree line massif height = 1
	//	    / \ |/  \ |
	//	   0   1|3   4| MMR INDICES
	//	   0 . 1|2 . 3| LEAF INDICES
	//     -----|-----|
	//       0  |  1  | MASSIF INDICES
	//
	// +--------+---++---+---+---+---+
	// | [1, 3] | 2 || 3 | 4 | 5 | 6 |
	// +--------+---++---+-------+---+
	//
	// When we add(4), we will add 5 geting local (3) then get(2) from the stack to create 6
	// The stack position we need is always top - (adding height - massif height)

	mc0 := mc
	//mc0.Data = append([]byte(nil), mc0.Data...)

	// simulate read by just un-marshaling the start from the data, which is currently the massif 0 data
	err = mc.Start.UnmarshalBinary(mc.Data)
	assert.Nil(t, err)

	// Now commit to the new massif
	err = mc.StartNextMassif()
	assert.Nil(t, err)

	// +--------+---++ check MI, FI are correct in the start header
	// | [1, 3] | 2 ||
	// +--------+---++
	assert.Equal(t, mc.Start.MassifIndex, uint32(1))
	assert.Equal(t, mc.Start.FirstIndex, uint64(3))

	// require exactly one entry in the new peak stack
	assert.Equal(t, mc.Start.PeakStackLen, uint64(1))

	// Check the stack has the expected value of mmr index 2 from massif 0's context
	assert.Equal(t, getFromStack(mc, 0), getFromData(mc0, 2))

	// fill massif 1, noting that there is a single extra node above the tree line
	// mc.Data = tc.padWithLeafEntries(mc.Data, 1<<MassifHeight-1+1)
	mc.Data = tc.PadWithNumberedLeaves(mc.Data, int(mc.Start.FirstIndex), 1<<massifHeight-1+1)

	// --- massif 2

	// We begin with the data of massif 1 from above
	//
	// +--------+---++---+---+---+---+
	// | [1, 3] | 2 || 3 | 4 | 5 | 6 |
	// +--------+---++---+-------+---+
	//
	//    stackLen(1) = 1
	//    popLen(1)   = 1
	//    pop stack   = stack[:stackLen-popLen] = stack[:1-1] = stack[:0]
	//    push stack  = append(stack, 6) (last leaf of massif 1)
	//
	// Massif 2 will look like this
	//
	//	2     \ 6
	//	      /\  \
	//	1    2  |  5  |  9  | --- massif tree line massif height = 1
	//	    / \ |/  \ | / \ |
	//	   0   1|3   4|7   8| MMR INDICES
	//	   0 . 1|2 . 3|4   5| LEAF INDICES
	//     -----|-----|-----|
	//       0  |  1  |  2  | MASSIF INDICES
	//
	// +--------+---++---+---+---+
	// | [2, 7] | 6 || 7 | 8 | 9 |
	// +--------+---++---+-------+
	//
	// When we add (9) we don't have enough nodes to build the next level so
	// massif 2 has no over flow, but it *must* carry forward the peak stack to
	// maintain the 'single blob look back' property.
	mc1 := mc
	// mc0.Data = append([]byte(nil), mc0.Data...)

	// simulate read by just un-marshaling the start from the data, which is currently the massif 0 data
	err = mc.Start.UnmarshalBinary(mc.Data)
	assert.Nil(t, err)

	// Now commit to the new massif
	err = mc.StartNextMassif()
	assert.Nil(t, err)

	// +--------+---++ check MI, FI are correct in the start header
	// | [2, 7] | 6 ||
	// +--------+---++
	assert.Equal(t, mc.Start.MassifIndex, uint32(2))
	assert.Equal(t, mc.Start.FirstIndex, uint64(7))

	// require exactly one entry in the new peak stack
	assert.Equal(t, mc.Start.PeakStackLen, uint64(1))

	// Check the stack has the expected value in mmr index 6 from massif 1's 4rth entry
	assert.Equal(t, getFromStack(mc, 0), getFromData(mc1, 6-mc1.Start.FirstIndex))

	// fill massif 2, noting that this time, unlike for massif 1, there are no nodes above the tree line
	// mc.Data = tc.padWithLeafEntries(mc.Data, 1<<MassifHeight-1)
	mc.Data = tc.PadWithNumberedLeaves(mc.Data, int(mc.Start.FirstIndex), 1<<massifHeight-1)

	// --- massif 3

	// We begin with the data of massif 2
	//
	// +--------+---++---+---+---+
	// | [2, 7] | 6 || 7 | 8 | 9 |
	// +--------+---++---+-------+
	//
	// stackLen(2) = 1
	// popLen(2)   = 0 (first example where we retain a full non-empty stack from the previous massif)
	// pop stack   = stack[:stackLen-popLen] = stack[:1-0] = stack[:1]
	// push stack  = append(stack, 9) (last leaf of massif 2)

	// Massif 3 will look like this
	//
	//                \14
	//           \  /  \ \
	//            \/    \ \
	//            /\     \ \
	//	2     \  6  \     \ 13
	//	      /\  \  \    /\  \
	//	1    2  |  5  |  9  |  \    | --- massif tree line massif height = 1
	//	    / \ |/  \ | / \ |  12   |
	//	   0   1|3   4|7   8|  /  \ | MMR INDICES
	//	   0 . 1|2 . 3|4   5|10   11| LEAF INDICES
	//     -----|-----|-----|-------|
	//       0  |  1  |  2  |    3  | MASSIF INDICES

	// +--------+---+---++----+----+----+----+----+
	// | [3,10] | 6 | 9 || 10 | 11 | 12 | 13 | 14 |
	// +--------+---+---++----+----+----+----+----+

	//
	// When we add (9) we don't have enough nodes to build the next level so
	// massif 2 has no over flow, but it *must* carry forward the peak stack to
	// maintain the 'single blob look back' property.
	mc2 := mc
	// mc2.Data = append([]byte(nil), mc0.Data...)
	// simulate read by just un-marshaling the start from the data, which is currently the massif 0 data
	err = mc.Start.UnmarshalBinary(mc.Data)
	assert.Nil(t, err)

	// Now commit to the new massif
	err = mc.StartNextMassif()
	assert.Nil(t, err)

	// +--------+---+---++ check MI, FI are correct in the start header
	// | [3, 10]| 6 | 9 ||
	// +--------+---+---++
	assert.Equal(t, mc.Start.MassifIndex, uint32(3))
	assert.Equal(t, mc.Start.FirstIndex, uint64(10))

	// require exactly two entries in the new peak stack
	assert.Equal(t, mc.Start.PeakStackLen, uint64(2))

	// Check the stack has the expected value of mmr indices 6 and 9 from massif 1's context
	assert.Equal(t, getFromStack(mc, 0), getFromData(mc1, 6-mc1.Start.FirstIndex))
	assert.Equal(t, getFromStack(mc, 1), getFromData(mc2, 9-mc2.Start.FirstIndex))

	// fill massif 3, noting that this time, as we hit a perfect power of two mmr size we gain a whole MMR tree level
	// mc.Data = tc.padWithLeafEntries(mc.Data, 1<<MassifHeight-1+2)
	mc.Data = tc.PadWithNumberedLeaves(mc.Data, int(mc.Start.FirstIndex), 1<<massifHeight-1+2)

	// --- massif 4
	//
	// Note that this case is particularly interesting because it completes a
	// full cycle from one perfect power to the next. massif 0 and massf 3 both
	// hit 'perfect' mmr trees. And the massif imediately after will see the
	// stack from the previous completely reset. It is a fact of the MMR
	// construction that the look back is never further than the most recent
	// 'perfect' tree completing massif. This creates a a very predictable and
	// very low growth rate for the ancestor stack we need to maintain. It grows
	// with the base 2 log of the height *above* the massif tree line. Which it self is traded off against the size of the mmr blobs
	//
	// We begin with Massif 3 from above
	//
	// +--------+---+---++----+----+----+----+----+
	// | [3,10] | 6 | 9 || 10 | 11 | 12 | 13 | 14 |
	// +--------+---+---++----+----+----+----+----+
	//
	// stackLen(3) = 2
	// popLen(3)   = 2 (first example where we *discard* all nodes on a 'not empty' stack at once)
	// pop stack   = stack[:stackLen-popLen] = stack[:2-2] = stack[:2]
	// push stack  = append(stack, 14) (last leaf of massif 3 and the perfect MMR root at that time)
	//
	//  3             \14
	//              /  \ \
	//            \/    \ \
	//            /\     \ \
	//	2     \  6  \     \ 13
	//	      /\  \  \    /\  \
	//	1    2  |  5  |  9  |  \    |  17  | --- massif tree line massif height = 1
	//	    / \ |/  \ | / \ |  12   | /  \ |
	//	   0   1|3   4|7   8|  /  \ |15  16| MMR INDICES
	//	   0 . 1|2 . 3|4   5|10   11|8    9| LEAF INDICES
	//     -----|-----|-----|-------|------|
	//       0  |  1  |  2  |    3  |   4  | MASSIF INDICES
	//
	// +--------+---++----+----+----+
	// | [4,15] | 14|| 15 | 16 | 17 |
	// +--------+---++----+----+----+

	mc3 := mc
	// mc2.Data = append([]byte(nil), mc0.Data...)
	// simulate read by just un-marshaling the start from the data, which is currently the massif 0 data
	err = mc.Start.UnmarshalBinary(mc.Data)
	assert.Nil(t, err)

	// Now commit to the new massif
	err = mc.StartNextMassif()
	assert.Nil(t, err)

	// +--------+---++ check MI, FI are correct in the start header
	// | [4, 15]| 14||
	// +--------+---++
	assert.Equal(t, mc.Start.MassifIndex, uint32(4))
	assert.Equal(t, mc.Start.FirstIndex, uint64(15))

	// require exactly one entry in the new peak stack
	stackLen := mmr.LeafMinusSpurSum(uint64(mc.Start.MassifIndex))
	assert.Equal(t, uint64(1), stackLen)
	assert.Equal(t, uint64(1), mc.Start.PeakStackLen)

	// Check the stack has the expected value of mmr index 14 from massif 3's content
	assert.Equal(t, getFromStack(mc, 0), getFromData(mc3, 14-mc3.Start.FirstIndex))

	// fill massif 4, noting that this time, as we hit a perfect power of two mmr size we gain a whole MMR tree level
	// mc.Data = tc.padWithLeafEntries(mc.Data, 1<<MassifHeight-1)
	mc.Data = tc.PadWithNumberedLeaves(mc.Data, int(mc.Start.FirstIndex), 1<<massifHeight-1)
}

// massifHeight := uint64(1)
//	3            14                             29
//	           /    \                                \
//	          /      \                     /          \
//	         /        \                   /            \
//	2      6 .      .  13                21            28
//	      /   \       /   \             / . \         /   \
//	1    2 --  5 --  9 --- 12  -----  17 -- 20  --- 24 --  27 ---- massif tree line
//	    / \  /  \   / \    /  \      /  \   / \     / \ . ./ \
//	   0   1 3   4  7   8 10   11  15   16 18  19  22  23 25  26 MMR INDICES
//	   0 . 1 2 . 3  4   5  6    7 . 8 .  9 10  11  12  13 14  15 LEAF INDICES

/*

    0: - r0 delta 0
    1: [0] r2 delta 1
    2: - r3 delta 0
    3: [2, 3] r6 delta 2
    4: - r7 delta 0
    5: [7] r9 delta 1
    6: - r10 delta 0
    7: [9, 10] r14 delta 3
    8: - r15  delta 0
    9: [15] r17 delta 1
   10: - r18 delta 0
   11: [18, 17] r21 delta 2
   12: - r22 delta 0
   13: [22] r24 delta 1
   14: - r14 delta 0
   15: [25, 24, 21, 14] r30 delta 4

    0: - r0
    1: [0] r2
    2: [0, 2] r3
    3: [0, 2, 3] r6
    4: [0, 2, 3, 6] r7
    5: [0, 2, 3, 6, 7] r9
    6: - r10
    7: [9, 10] r14
    8: - r15
    9: [15] r17
   10: - r18
   11: [18, 17] r21
   12: - r22
   13: [22] r24
   14: - r14
   15: [25, 24, 21, 14] r30

    0: [] - r0 push (0) pop x0
    1: [0] push (2) pop x1 -> [2]
    2: [0, 2] pus(3) pop x1 -> [2, 3]
    3: [0, 2, 3] push(6) pop x2 -> [6]
    4: [0, 2, 3, 6] push(7) pop x0 -> [6, 7]
    5: [0, 2, 3, 6, 7] push(9) pop x1 -> [6, 9]
    6: - r10  push (10) pop x0
    7: [9, 10] push(14) pop x3
    8: - r15 push(15) pop x0
    9: [15] push(17) pop x1
   10: - push(18) pop 0
   11: [18, 17] push(21) pop x2
   12: - push(22) pop x0
   13: [22] push(24) pop x1
   14: - push(14) pop x0
   15: [25, 24, 21, 14] push(30) pop x 4


*/
