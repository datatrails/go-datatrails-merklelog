package massifs

import (
	"context"
	"errors"

	"github.com/datatrails/go-datatrails-common/azblob"
	"github.com/datatrails/go-datatrails-common/logger"
	"github.com/datatrails/go-datatrails-merklelog/mmr"
)

var (
	ErrMassifNotFound          = errors.New("the requested massif blob is not found")
	ErrLocalAccessNotSupported = errors.New("this reader implementation does not provide local filesystem access")
)

type MassifReader struct {
	log   logger.Logger
	store LogBlobReader
	opts  ReaderOptions
}

func NewMassifReader(
	log logger.Logger, store LogBlobReader,
	opts ...ReaderOption,
) MassifReader {
	r := MassifReader{
		log:   log,
		store: store,
	}
	for _, o := range opts {
		o(&r.opts)
	}
	return r
}

func (mr *MassifReader) GetMassif(
	ctx context.Context, tenantIdentity string, massifIndex uint64,
	opts ...ReaderOption,
) (MassifContext, error) {

	options := ReaderOptionsCopy(mr.opts)
	for _, o := range opts {
		o(&options)
	}

	var err error
	mc := MassifContext{
		TenantIdentity: tenantIdentity,
		LogBlobContext: LogBlobContext{
			BlobPath: TenantMassifBlobPath(tenantIdentity, massifIndex),
		},
	}
	if err = mr.readAndPrepareContext(ctx, &mc, options.remoteReadOpts...); err != nil {
		return MassifContext{}, err
	}
	return mc, nil
}

func (mr *MassifReader) readAndPrepareContext(ctx context.Context, mc *MassifContext, opts ...azblob.Option) error {
	err := mc.ReadData(ctx, mr.store, opts...)
	if err != nil {
		return err
	}

	err = mc.Start.UnmarshalBinary(mc.Data)
	if err != nil {
		return err
	}
	if !mr.opts.noGetRootSupport {
		if err = mc.CreatePeakStackMap(); err != nil {
			return err
		}
	}
	return nil
}

func (mr *MassifReader) GetHeadMassif(
	ctx context.Context, tenantIdentity string,
	opts ...ReaderOption,
) (MassifContext, error) {

	options := ReaderOptionsCopy(mr.opts)
	for _, o := range opts {
		o(&options)
	}

	var err error
	blobPrefixPath := TenantMassifPrefix(tenantIdentity)

	mc := MassifContext{
		TenantIdentity: tenantIdentity,
	}
	var massifCount uint64
	mc.LogBlobContext, massifCount, err = LastPrefixedBlob(ctx, mr.store, blobPrefixPath, options.remoteListOpts...)
	if err != nil {
		return MassifContext{}, err
	}
	if massifCount == 0 {
		return MassifContext{}, ErrMassifNotFound
	}
	if err = mr.readAndPrepareContext(ctx, &mc, options.remoteReadOpts...); err != nil {
		return MassifContext{}, err
	}

	return mc, nil
}

// GetLazyContext reads the blob metadata of a logical blob but does _not_ read the data.
func (mr *MassifReader) GetLazyContext(
	ctx context.Context, tenantIdentity string, which LogicalBlob,
	opts ...ReaderOption,
) (LogBlobContext, uint64, error) {

	options := ReaderOptionsCopy(mr.opts)
	for _, o := range opts {
		o(&options)
	}

	blobPrefixPath := TenantMassifPrefix(tenantIdentity)

	var massifIndex uint64

	var err error
	var logBlobContext LogBlobContext
	switch which {
	case FirstBlob:
		logBlobContext, err = FirstPrefixedBlob(ctx, mr.store, blobPrefixPath, options.remoteListOpts...)
	case LastBlob:
		logBlobContext, massifIndex, err = LastPrefixedBlob(ctx, mr.store, blobPrefixPath, options.remoteListOpts...)
	}
	if err != nil {
		return LogBlobContext{}, 0, err
	}
	return logBlobContext, massifIndex, nil
}

func (mr *MassifReader) GetFirstMassif(
	ctx context.Context, tenantIdentity string,
	opts ...ReaderOption,
) (MassifContext, error) {

	options := ReaderOptionsCopy(mr.opts)
	for _, o := range opts {
		o(&options)
	}

	var err error
	blobPrefixPath := TenantMassifPrefix(tenantIdentity)

	mc := MassifContext{
		TenantIdentity: tenantIdentity,
	}
	mc.LogBlobContext, err = FirstPrefixedBlob(ctx, mr.store, blobPrefixPath, options.remoteListOpts...)
	if err != nil {
		return MassifContext{}, err
	}
	if err = mr.readAndPrepareContext(ctx, &mc, options.remoteReadOpts...); err != nil {
		return MassifContext{}, err
	}

	return mc, nil
}

// MassifIndexFromLeafIndex gets the massif index of the massif that the given leaf is stored in,
//
//	given the leaf index of the leaf.
//
// This is found with the given massif height, which is constant for all massifs.
func MassifIndexFromLeafIndex(massifHeight uint8, leafIndex uint64) uint64 {

	// first find how many leaf nodes each massif can hold.
	//
	// Note: massifHeight starts at index 1, whereas height index for HeighIndexLeafCount starts at 0.
	massifMaxLeaves := mmr.HeightIndexLeafCount(uint64(massifHeight) - 1)

	// now find the massif.
	//
	// for context, see: https://github.com/datatrails/epic-8120-scalable-proof-mechanisms/blob/main/mmr/forestrie-mmrblobs.md#blob-size
	//
	// Note: massif indexes start at 0.
	// Note: leaf indexes starts at 0.
	//
	// Therefore, given a massif height of 2, that has max leaves of 4;
	//  if a leaf index of 3 is given, then it is in massif 0, along with leaves, 0, 1 and 2.
	return leafIndex / massifMaxLeaves

}

// MassifIndexFromMMRIndex gets the massif index of the massif that the given leaf is stored in
//
//	given the mmr index of the leaf.
//
// NOTE: if the mmrIndex is not a leaf node, then error is returned.
func MassifIndexFromMMRIndex(massifHeight uint8, mmrIndex uint64) uint64 {

	leafIndex := mmr.LeafIndex(mmrIndex)

	return MassifIndexFromLeafIndex(massifHeight, leafIndex)

}
