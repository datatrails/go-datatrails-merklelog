package mmrblobs

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrGetIndexUnavailable  = errors.New("requested mmr index not available")
	ErrAncestorStackInvalid = errors.New("the ancestor stack is invalid due to bad header information")
)

// MassifContext enables retrieving the tenants log from blob storage.
//
//
// It is constructed entirely from data held in the massif blob and the blob
// imediately prior to it. Given the blob itself and only the 'tail nodes' from
// the preceding blob, it is possible to generate proofs without knowlege of any
// further blobs.
//
// Massif blobs are defined by the _fixed_ number of _leaves_ they contain. We
// require that count to be a power of 2 and > 1. Given that, the number of
// nodes in a massif is just: n + n - 1. This follows from the binary nature of
// the tree.
//
// For example, with n leaves = 4 we get:  4 + 3 = 7
//
// This is the corresponding 'position' tree, with indication of how the MMR is
// 'chunked' into sub mountain ranges which we call 'massifs'
//
//	3        \   15   massif 1 \
//	          \/    \           \
//	 massif 0 /\     \           |    'alpine zone' is above the massif tree line
//	         /   \    \          |
//	2 ..... 7.....|....14........|...... 22 ..... Masif Root Index identifies the massif root
//	      /   \   |   /   \      |      /
//	1    3     6  | 10     13    |    18     21
//	    / \  /  \ | / \    /  \  |   /  \
//	   1   2 4   5| 8   9 11   12| 16   17 19 20
//	   0   1 3   4| 7   8 10   11| 15   16 18 19
//	   | massif 0 |  massif 1 .  | massif 2 ....>

//	1 << 3 - 1 << 2 = 8 - 4 = 4
//	1 << 4 - 1 << 3 = 16 - 8 = 8
//
// Massif Root Index                7-1 |       8+7-2  |              16 + 7-2
// Massif Last Leaf Index           5-1 |       8+5-2  |              16 + 5-2
//
// In order to require the power 2 property for the leaf count, we configure the
// massif size by its 'height'. Here, our 4 leaf tree has height 3 (level index 2)
//
// So typically instead of n + n -1, where n is the massif leaf count we instead do
//
// Massif Root Index      = (1 << h) - 2
// Massif Last Leaf Index = (1 << h) - h - 1
type MassifContext struct {
	TenantIdentity string
	BlobPath       string
	Tags           map[string]string
	ETag           string
	LastRead       time.Time
	LastModfified  time.Time

	// Read from the first log entry in the blob. If Creating is true and Found
	// > 0, this is the Start header of the *previous* massif
	Start MassifStart
	Data  []byte

	// the following properties are for dealing with addition of the last leaf
	// in the massif they are only valid during the call to AddHashedLeaf which
	// appends the last leaf of the massif (other appends are guaranteed not to
	// reference nodes from earlier massif blobs)

	// Set to the peak stack index containing the *next* ancestor node that will
	// be needed. Initialised in AddLeafHash and only valid during that call
	nextAncestor int
}

// Get returns the value associated with the node at MMR index i
//
// Note that due to the structure of the MMR we are guaranteed that adding a
// node will only reference other nodes in the *current* massif, OR it will
// reference the root of the previous massif. As we link the massif blobs by
// including the root of the previous massif as the value for the first massif
// entry, we can return it directly. Eg in fhe following, the left child of
// position 15 is the root of massif 0 at position 7, and similarly, the left
// child of the root of massif 2 will be position 15. As Get works in indices,
// that will be indices 14 and 6.
//
//	3        \   15   massif 1 \ . massif 2
//	          \/    \           \
//	 massif 0 /\     \           |
//	         /   \    \          |
//	2 ..... 7.....|....14........|...... 22 .....
//	      /   \   |   /   \      |      /
//	1    3     6  | 10     13    |    18     21
//	    / \  /  \ | / \    /  \  |   /  \
//	   1   2 4   5| 8   9 11   12| 16   17 19 20
//	   0   1 3   4| 7   8 10   11| 15   16 18 19
//	   | massif 0 |  massif 1 .  | massif 2 ....>
//
// This method satisfies the Get method of the MMR NodeAdder interface
func (mc MassifContext) Get(i uint64) ([]byte, error) {

	// Normal case, reference to a node included in the current massif
	if i >= mc.Start.FirstIndex {
		return IndexedLogValue(mc.Data[mc.LogStart():], i-mc.Start.FirstIndex), nil
	}

	// Ok, its a reference to the root of the previous massif or this is an error case

	if mc.Start.FirstIndex == 0 {
		return nil, fmt.Errorf("%w: the first massif has no ancestors", ErrGetIndexUnavailable)
	}

	// The ancestor stack is maintained so that the nodes we need are listed in
	// the order they will be asked for. And we initialise nextAncestor in
	// AddLeafHash to the top of the stack

	if mc.nextAncestor < 0 {
		return nil, fmt.Errorf("%w: exceeded the nodes included in the ancestor peak stack, requesting %d", ErrGetIndexUnavailable, i)
	}

	stackTop := mc.LogStart()
	stackStart := mc.PeakStackStart()
	if stackStart > stackTop {
		return nil, fmt.Errorf("%w: invalid context, requesting %d", ErrAncestorStackInvalid, i)
	}
	endOffset := (ValueBytes * (uint64(mc.nextAncestor) + 1))
	if endOffset > (stackTop - stackStart) {
		return nil, fmt.Errorf("%w: exceeded the data range of the ancestor peak stack, requesting %d", ErrAncestorStackInvalid, i)
	}
	valueStart := stackTop - endOffset
	mc.nextAncestor -= 1
	return mc.Data[valueStart : valueStart+ValueBytes], nil
}

func (mc MassifContext) FixedHeaderEnd() uint64 {
	return ValueBytes
}

func (mc MassifContext) IndexHeaderStart() uint64 {
	return mc.FixedHeaderEnd()
}

// IndexHeaderEhd returns the end of the bytes reserved for the index header.
// Currently, nothing is stored in this.
// XXX: TODO: Consider removing the field all together
func (mc MassifContext) IndexHeaderEnd() uint64 {
	return mc.IndexHeaderStart() + IndexHeaderBytes
}

// IndexStart returns the index of the first **byte** of index data.
func (mc MassifContext) IndexStart() uint64 {
	return mc.IndexHeaderEnd()
}

func (mc MassifContext) IndexLen() uint64 {
	return (1 << mc.Start.MassifHeight)
}

func (mc MassifContext) IndexSize() uint64 {
	return mc.IndexLen() * IndexEntryBytes
}

// IndexEnd returns the byte index of the end of index data
func (mc MassifContext) IndexEnd() uint64 {
	return mc.IndexStart() + IndexEntryBytes*(1<<mc.Start.MassifHeight)
}

func (mc MassifContext) PeakStackStart() uint64 {
	return mc.IndexEnd()
}

func (mc MassifContext) LogStart() uint64 {
	return mc.IndexEnd() + ValueBytes*mc.Start.PeakStackLen
}

// Count returns the number of log entries in the massif
func (mc MassifContext) Count() uint64 {
	logStart := mc.LogStart()
	if logStart > uint64(len(mc.Data)) {
		return (uint64(len(mc.Data)) - logStart) / LogEntryBytes
	}
	return (uint64(len(mc.Data)) - logStart) / LogEntryBytes
}

// RangeCount returns the total number of log entries in the MMR upto and including this context
func (mc MassifContext) RangeCount() uint64 {
	return mc.Start.FirstIndex + mc.Count()
}

// TreeRootIndex returns the root index for the tree with height
func TreeRootIndex(height uint8) uint64 {
	return (1 << height) - 2
}

// RangeRootIndex return the Massif root node's index in the overall MMR  given
// the massif height and the first index of the MMR it contains
func RangeRootIndex(firstIndex uint64, height uint8) uint64 {
	return firstIndex + (1 << height) - 2
}

// RangeLastLeafIndex returns the mmr index of the last leaf given the first
// index of a massif and its height.
func RangeLastLeafIndex(firstIndex uint64, height uint8) uint64 {
	return firstIndex + TreeLastLeafIndex(height)
}

// TreeLastLeafIndex returns the index of the last leaf in the tree with the
// given height (1 << h) - h -1 works because the number of nodes required to
// include the last leaf is always equal to the MMR height produced by that
func TreeLastLeafIndex(height uint8) uint64 {
	return (1 << height) - uint64(height) - 1
}

// TreeSize returns the maximum byte size of the tree based on the defined log
// entry size
func TreeSize(height uint8) uint64 {
	return TreeCount(height) * LogEntryBytes
}

// MaxCount returns the node count
func TreeCount(height uint8) uint64 {
	return ((1 << height) - 1)
}
