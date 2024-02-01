package mmrblobs

const (
// .         | 0   |           |  21 - 22 | 23   26|27         27| 28 -  31 |

)

// IndexHeader exists to keep track of the number of leaves represented by the
// mmr data.
//
// Background:
//
// By keeping the index and the log together, we guarantee mutual consistency -
// provided the log and the idex values are correctly calculated, a single write
// commits the change back to the blob store.
//
// Because the data is combined, we can't use file size as a proxy for the
// membership count.
//
// Regardless of whether we pre-allocate the index data or whether we accumulate
// it as we do the mmr, we need to know how many leaves are in the index.  An
// algorithm to derive a leaf index form an MMR position exists, it is sub
// linear but a bit fiddly to get right.
//
// At least for now, we are going to explicitly track the count of leaves in a
// counter value in the blob.
type IndexHeader struct {
	Index uint64
}
