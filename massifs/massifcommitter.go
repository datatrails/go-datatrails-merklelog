package massifs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/datatrails/go-datatrails-common/azblob"
	"github.com/datatrails/go-datatrails-common/logger"
)

type MassifCommitter struct {
	Cfg   MassifCommitterConfig
	Log   logger.Logger
	Store massifStore
}

type MassifCommitterConfig struct {
	CommitmentEpoch uint32
}

func NewMassifCommitter(cfg MassifCommitterConfig, log logger.Logger, store massifStore) *MassifCommitter {
	c := &MassifCommitter{
		Cfg:   cfg,
		Log:   log,
		Store: store,
	}
	return c
}

func (c *MassifCommitter) CommitContext(ctx context.Context, mc MassifContext) (*azblob.WriteResponse, error) {

	var err error

	// Note that while we are continually overwriting the blob, on the period
	// cadence we will be publishing whatever its current mmr root is to some
	// thing we cant change (public block chain or at least our own private
	// ledger). So if we ever break the append only rule, it will be evident
	// (and not good).

	opts := []azblob.Option{azblob.WithTags(mc.Tags)}
	// CRITICAL: we _must_ use the etag to gaurd against racy updates. It will be absent only when crating the blob
	if mc.ETag != "" {
		opts = append(opts, azblob.WithEtagMatch(mc.ETag))
	} else {
		if mc.Creating == false {
			return nil, errors.New("etag is required when updating any blob")
		}
	}
	// Also CRITICAL: We must set the not-exists option if we are creating a new
	// blob. so we don't racily overwrite a new blob
	if mc.Creating {
		// The way to spell 'fail without modifying if the blob exists' is to require that no blob matches *any* etag.
		opts = append(opts, azblob.WithEtagNoneMatch("*"))
	}

	wr, err := c.Store.Put(ctx, mc.BlobPath, azblob.NewBytesReaderCloser(mc.Data),
		opts...,
	)
	if err != nil {
		return wr, err
	}

	// XXX: TODO caching to merklebuilder
	// // Note: we add with the *new* etag
	// c.lastRead.Add(*rr.ETag, *rr.LastModified, mc)

	return wr, err
}

func (c *MassifCommitter) createFirstMassifContext(
	tenantIdentity string, massifHeight uint8) (MassifContext, error) {

	// XXX: TODO: we _could_ just roll an id so that we never need to deal with
	// the zero case. for the first blob that is entirely benign.
	start := NewMassifStart(0, c.Cfg.CommitmentEpoch, massifHeight, 0, 0)

	// the zero values, or those explicitly set above are correct
	data, err := start.MarshalBinary()
	if err != nil {
		return MassifContext{}, err
	}

	mc := MassifContext{
		TenantIdentity: tenantIdentity,
		Creating:       true,
		LogBlobContext: LogBlobContext{
			BlobPath: TenantMassifBlobPath(tenantIdentity, 0),
			Tags:     map[string]string{},
		},
		// epoch, massifIndex and firstIndex are zero and prev root is 32 bytes of zero
		Start: start,
	}
	// We pre-allocate and zero-fill the index, see the commentary in StartNextMassif
	mc.Data = append(data, mc.InitIndexData()...)

	// mc.FirstIndex zero value is correct
	SetFirstIndex(mc.Start.FirstIndex, mc.Tags)

	return mc, nil
}

// GetCurrentContext gets the current mmr blob context for the tenant
//
// The returned context is ready to accept new log entries.
func (c *MassifCommitter) GetCurrentContext(
	ctx context.Context, tenantIdentity string, massifHeight uint8) (MassifContext, error) {

	// There are 3 states to consider here
	// 1. No blobs exist -> setup context for creating first blob
	// 2. A previous full blob exists -> setup context for creating a new blob
	// 3. The most recent blob is not full -> setup context for extending current blob

	var err error

	mc, massifCount, err := c.GetLastMassif(ctx, tenantIdentity)
	if err != nil {
		return MassifContext{}, err
	}
	if massifCount == 0 {
		return c.createFirstMassifContext(tenantIdentity, massifHeight)
	}

	// XXX: TODO: consider the use of the etag here. I'm using it just because I
	// think it will avoid confusing triage situations list vs get. It shouldn't
	// actually matter at this point.

	// If we are creating, we need to read the bytes from the previous blob to
	// be able to make the first mmr entry from the root of the last massif.

	var rr *azblob.ReaderResponse
	rr, mc.Data, err = c.cachedBlobRead(
		ctx, mc.BlobPath, azblob.WithEtagMatch(mc.ETag), azblob.WithGetTags())
	if err != nil {
		return mc, err
	}

	// All valid massifs are created with at least the single fixed (versioned)
	// header record.
	err = mc.Start.UnmarshalBinary(mc.Data)
	if err != nil {
		return mc, err
	}

	mc.Tags = rr.Tags

	// NOTICE: While the *index* on blob tags is eventually consistent, the tags
	// read directly with the blob are *guaranteed* by azure to be 'the values
	// last written'. This is a critical assumption for our crash fault
	// tolerant model.
	//
	// "After you set your index tags, they exist on the blob and can be
	// retrieved immediately.  It might take some time before the blob index
	// updates." -- https://learn.microsoft.com/en-us/azure/storage/blobs/storage-manage-find-blobs?tabs=azure-portal
	firstIndex, err := GetFirstIndex(mc.Tags)
	if err != nil {
		return mc, err
	}
	if firstIndex != mc.Start.FirstIndex {
		return mc, fmt.Errorf(
			"%w: %x vs %x",
			ErrIncorrectFirstIndexTag,
			firstIndex, mc.Start.FirstIndex)
	}

	// The current first & last is initialized from what we read

	mc.LastModified = *rr.LastModified
	mc.LastRead = time.Now()

	// If the blob has space for more nodes, the context is ready and we have
	// all the state setup.  case 3: existing blob with space, !creating.
	//  This works because no matter which massif blob this is, just prior to
	// adding the last *leaf*, the occupied size will be less than the massif
	// base size. And adding the leaf and its necessary interior nodes will
	// immediately exceed or equal the base size configured for a massif.
	sz := TreeSize(mc.Start.MassifHeight)
	start := mc.LogStart()
	if uint64(len(mc.Data))-start < sz {
		return mc, nil
	}

	// if the previous is complete, attempt to start a new massif
	mc.Creating = true
	mc.ETag = ""
	mc.LastModified = time.UnixMilli(0)
	mc.LastRead = time.UnixMilli(0)

	mc.BlobPath = TenantMassifBlobPath(tenantIdentity, massifCount)

	// re-create Start for the new blob

	return mc, mc.StartNextMassif()
}

// GetLastMassif finds the most recently created massif blob for the tenant and
// returns its id. A massif's id is just 1+ its zero based index in the tenants
// list of mmr blobs. A return value of 0 means no blobs exist for the tenant
func (c *MassifCommitter) GetLastMassif(
	ctx context.Context, tenantIdentity string) (MassifContext, uint64, error) {

	mc := MassifContext{
		TenantIdentity: tenantIdentity,
	}

	blobPrefixPath := TenantMassifPrefix(tenantIdentity)

	bc, massifCount, err := LastPrefixedBlob(
		ctx, c.Store, blobPrefixPath)
	if err != nil {
		return mc, massifCount, err
	}
	mc.ETag = bc.ETag
	mc.LastModified = bc.LastModified
	mc.BlobPath = bc.BlobPath

	return mc, massifCount, nil
}

// cachedBlobRead reads the blob with potential caching and returns the
// underlying azblob response as the most consistent way to propagate the blob
// metadata to the caller. Note that on return, regardless of error the reader
// is nil as it has been completely exhausted or otherwise disposed of.
func (c *MassifCommitter) cachedBlobRead(
	ctx context.Context, blobPath string, opts ...azblob.Option) (*azblob.ReaderResponse, []byte, error) {
	return BlobRead(ctx, blobPath, c.Store, opts...)
}
