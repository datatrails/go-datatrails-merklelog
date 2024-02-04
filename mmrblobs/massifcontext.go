package mmrblobs

import (
	"errors"
	"fmt"
	"hash"

	"github.com/datatrails/go-datatrails-merklelog/mmr"
)

var (
	ErrGetIndexUnavailable      = errors.New("requested mmr index not available")
	ErrMassifFull               = errors.New("the current massif is full")
	ErrAncestorStackUnderfilled = errors.New("the ancestor stack data is to short to be valid")
	ErrAncestorStackInvalid     = errors.New("the ancestor stack is invalid due to bad header information")
	ErrMissingPrevBlobLastID    = errors.New("expected snowflake id carry from previous blob not available")
)

// MassifContext enables appending to the log
//
// The returned context is ready to accept new log entries.
//
// It is constructed entirely from data held in the massif blob and the blob
// immediately prior to it. Given the blob itself and only the 'tail nodes' from
// the preceding blob, it is possible to extend the log without knowledge of any
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
	LogBlobContext
	TenantIdentity string

	// This context deals with the three different massif states:
	// 1. no blobs exist                                   -> creating = true
	// 2. a previous full blob exists, starting a new blob -> creating = true
	// 3. the most recent blob is not full                 -> creating = false
	Creating bool

	// Read from the first log entry in the blob. If Creating is true and Found
	// > 0, this is the Start header of the *previous* massif
	Start MassifStart

	// the following properties are for dealing with addition of the last leaf
	// in the massif they are only valid during the call to AddHashedLeaf which
	// appends the last leaf of the massif (other appends are guaranteed not to
	// reference nodes from earlier massif blobs)

	// Set to the peak stack index containing the *next* ancestor node that will
	// be needed. Initialized in AddLeafHash and only valid during that call
	nextAncestor int

	// lastIDPreviousBlob preserves the last snowflake id from the previous
	// massif while we are adding the first entry. It is only used when adding
	// the first entry in a blob (when Creating is also true)
	lastIDPreviousBlob uint64
}

func (mc *MassifContext) StartNextMassif() error {
	// re-create Start for the new blob

	var err error
	// This enables a strict guarantee of uniqueness per log that is
	// enforced even in the face of the process restart + ip re-use + bad clocks
	// corner case. No matter what, we will never add an entry to an mmr
	// that isn't strictly greater than all others in the log.
	mc.lastIDPreviousBlob, err = mc.GetLastSnowflakeID()
	if err != nil {
		return err
	}

	// From here, mc.Start is logically the *previous* massif blob. And we start
	// the next massif based on the header of the previous.
	nextPeakStack, err := mc.NextPeakStack()
	if err != nil {
		return err
	}

	nextStart := NewMassifStart(
		mc.Start.Epoch, mc.Start.MassifHeight,
		// Note: at this point mc.Start and mc.Data refer to the *previous*
		// massif blob, so we can use it to compute the first index of the new
		// blob we are about to create.
		mc.Start.MassifIndex+1, mc.RangeCount())
	SetFirstIndex(nextStart.FirstIndex, mc.Tags)
	nextData, err := nextStart.MarshalBinary()
	if err != nil {
		return err
	}

	// We pre-allocate zero filled data for the index. When the blob is
	// complete, the index will be fully populated. We store a trie key in it,
	// which provides for data recovery & additional proof types, and also the
	// minimal information we need to retain in order to update confirmation
	// status. The fixed increase on read size is expected to *improve*
	// performance: It turns out, according to the azure guidance, this should
	// actually make the blobs perform better.  If this causes the blob to be
	// greater than 256k, it will get placed in higher throughput storage from
	// the start.  See
	// https://learn.microsoft.com/en-us/azure/storage/blobs/storage-performance-checklist#partitioning
	nextData = append(nextData, mc.InitIndexData()...)

	// PeakStackLen is _not_ marshaled into the header, we can always compute it when needed
	nextStart.PeakStackLen = uint64(len(nextPeakStack) / ValueBytes)
	nextData = append(nextData, nextPeakStack...)

	// store the updated data and update the start configuration for the new stack
	mc.Start = nextStart
	mc.Data = nextData

	return nil
}

func (mc MassifContext) InitIndexData() []byte {
	return make([]byte, IndexHeaderBytes+mc.IndexSize())
}

// NextPeakStack accepts the peak stack from the previous massif and returns the
// start data and stack for the current massif start details.
func (mc MassifContext) NextPeakStack() ([]byte, error) {

	var err error

	// Remembering that the 'push' to the stack is always the last log entry so
	// we just leave it where it is naturally and gather it into the stack only
	// when we propagate the stack to the next massif here. And we need to do
	// that before we pop.
	peakStack, err := mc.GetAncestorPeakStack()
	if err != nil {
		return nil, err
	}
	// Note: we don't need to compute the stack length here, but it serves as a
	// good early detector for data corruption issues.
	stackLen := mmr.LeafMinusSpurSum(uint64(mc.Start.MassifIndex))
	if uint64(len(peakStack)/ValueBytes) != stackLen {
		return nil, fmt.Errorf("%w: computed stack length doesn't match accumulated stack length", ErrAncestorStackInvalid)
	}
	pop := mmr.SpurHeightLeaf(uint64(mc.Start.MassifIndex))

	// do the stack pop, the append happens naturally when the last leaf is added
	// due to our always collecting it from the end of the log (via GetPeakStack
	// above)
	peakStack = peakStack[:(stackLen-pop)*ValueBytes]

	// Now we have popped the ancestors we are done with, we can push the last
	// value from the previous massif.
	peakStack = append(peakStack, mc.GetLastValue()...)
	return peakStack, nil
}

// GetPeakStack returns the ancestor peak stack plus the last value of the
// current massif. This method should only be called on a complete massif. The
// caller is responsible for ensuring this condition is met.
func (mc MassifContext) GetPeakStack() ([]byte, error) {
	ancestors, err := mc.GetAncestorPeakStack()
	if err != nil {
		return nil, err
	}
	return append(ancestors, mc.GetLastValue()...), nil
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
func (mc *MassifContext) Get(i uint64) ([]byte, error) {

	// Normal case, reference to a node included in the current massif
	if i >= mc.Start.FirstIndex {
		return IndexedLogValue(mc.Data[mc.LogStart():], i-mc.Start.FirstIndex), nil
	}

	// Ok, its a reference to the root of the previous massif or this is an error case

	if mc.Start.FirstIndex == 0 {
		return nil, fmt.Errorf("%w: the first massif has no ancestors", ErrGetIndexUnavailable)
	}

	// The ancestor stack is maintained so that the nodes we need are listed in
	// the order they will be asked for. And we initialize nextAncestor in
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

// Append adds the leaf value to the log and returns the MMR index of the _next_ node
// This method satisfies the Append method of the MMR NodeAdder interface
func (mc *MassifContext) Append(value []byte) (uint64, error) {

	if len(value) != ValueBytes {
		return 0, ErrLogValueBadSize
	}

	// XXX: TODO: ideally we would check for over flow here. But it is awkward
	// and log base 2 n to work out the actual limit of this context. If we want
	// that, we would capture it in GetCurrentContext The add leaf method
	// pre-flight checks on the highest leaf index which can be computed
	// directly at any time. Over flow after that check is only possible if our
	// basic mmr add is bust and that is extensively covered by unit tests.

	mc.Data = append(mc.Data, value...)
	return mc.RangeCount(), nil
}

// AddHashedLeaf adds the leaf value and corresponding index data to the log and
// index. On error, the current data buffer should be discarded entirely (not
// written back to storage)
//
// Returns the resulting size of the mmr if the leaf is addes successfully.
func (mc *MassifContext) AddHashedLeaf(hasher hash.Hash, index []byte, value []byte) (uint64, error) {

	if len(value) != ValueBytes {
		return 0, ErrLogValueBadSize
	}
	if len(index) != IndexEntryBytes {
		return 0, ErrIndexEntryBadSize
	}

	count := mc.Count()
	iLast := mc.LastLeafIndex()

	if mc.Start.FirstIndex+count > iLast {
		return 0, ErrMassifFull
	}

	if mc.Start.FirstIndex+count == iLast {
		mc.nextAncestor = int(mc.Start.PeakStackLen) - 1
	}

	// Overwrite the pre-allocated index entry with the index data.  The index
	// entry index is the count we have before adding the leaf.
	indexEntryOffset := mc.IndexStart() + count*IndexEntryBytes
	copy(mc.Data[indexEntryOffset:indexEntryOffset+IndexEntryBytes], index)

	// Note: assume that the whole update is discarded on error, including the index update above.

	// Returns the new MMR size if the new leaf is added successfully
	return mmr.AddHashedLeaf(mc, hasher, value)
}

// GetAncestorPeakStack returns the stack of ancestor peaks accumulated and
// retained from previous massifs. These are all the nodes that will be (or
// were) referenced when adding the last leaf to the current massif. Note that
// when carrying this stack forward to the next massif header, the last leaf is
// considered to have been 'pushed' on the stack and should be copied forward as
// the new accumulated stack head.
func (mc MassifContext) GetAncestorPeakStack() ([]byte, error) {

	peakStackStart := mc.PeakStackStart()
	logStart := mc.LogStart()
	if peakStackStart == logStart {
		return nil, nil
	}

	// It must be empty or have room for at least one item
	if peakStackStart+ValueBytes > logStart {
		return nil, fmt.Errorf("%w: peakStackEnd + entry size > logStart:  %d > %d", ErrAncestorStackInvalid, peakStackStart+ValueBytes, logStart)
	}

	// Must be properly aligned
	if (logStart-peakStackStart)%ValueBytes != 0 {
		return nil, fmt.Errorf("%w: size %% entry size=%d", ErrAncestorStackInvalid, (logStart-peakStackStart)%ValueBytes)
	}

	if mc.Data == nil {
		return nil, fmt.Errorf("%w: no data available", ErrAncestorStackInvalid)
	}

	return mc.Data[peakStackStart:logStart], nil
}

// GetLastSnowflakeID returns the snowflake of the last entry in the log
func (mc MassifContext) GetLastSnowflakeID() (uint64, error) {

	leafCount := mc.MassifLeafCount()

	if leafCount == 0 {
		// The count can only be zero when we are creating and so adding the
		// first entry. A special arrangement in StartNextMassif squirrels away
		// the last snowflake id of the previous blob before we discard its
		// data.
		// For the very first blob, this value will be zero and that is less
		// than all timestamps as it does not include machine or sequence data.
		return mc.lastIDPreviousBlob, nil
	}

	offset := mc.IndexStart() + (leafCount-1)*IndexEntryBytes
	return GetIndexSnowflakeID(mc.Data, offset), nil
}

// MassifLeafCount returns the number of leaves in the current blob (If you want
// the number of leaves in the entire mmr call mmr.LeafCount directly)
func (mc MassifContext) MassifLeafCount() uint64 {

	// Get the count of leaves in the entire mmr
	count := mmr.LeafCount(mc.RangeCount())
	// Subtract the number of leaves in the mmr defined by the end of the last blob
	// to get the count of leaves in the current blob
	return count - mmr.LeafCount(mc.Start.FirstIndex)
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

func (mc MassifContext) GetLastValue() []byte {
	if len(mc.Data) < ValueBytes {
		return nil
	}
	return mc.Data[len(mc.Data)-ValueBytes:]
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

// LastLeafIndex returns the leaf index for the last entry that can be added to
// the mmr. This is typically used to check if the last entry is being added.
func (mc MassifContext) LastLeafIndex() uint64 {
	return RangeLastLeafIndex(mc.Start.FirstIndex, mc.Start.MassifHeight)
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
