package mmrblobs

import (
	"context"
	"time"

	"github.com/datatrails/go-datatrails-common/azblob"
	"github.com/datatrails/go-datatrails-common/logger"
	"github.com/datatrails/go-datatrails-merklelog/mmr"
)

type massifReader interface {
	Reader(
		ctx context.Context,
		identity string,
		opts ...azblob.Option,
	) (*azblob.ReaderResponse, error)

	List(ctx context.Context, opts ...azblob.Option) (*azblob.ListerResponse, error)
}

type MassifReader struct {
	log   logger.Logger
	store massifReader
}

func NewMassifReader(log logger.Logger, store massifReader) *MassifReader {
	r := &MassifReader{
		log:   log,
		store: store,
	}
	return r
}

func (mr *MassifReader) GetMassif(
	ctx context.Context, tenantIdentity string, massifIndex uint64,
	opts ...azblob.Option,
) (MassifContext, error) {

	var err error
	var rr *azblob.ReaderResponse
	mc := MassifContext{
		TenantIdentity: tenantIdentity,
		BlobPath:       TenantMassifBlobPath(tenantIdentity, massifIndex),
	}

	rr, mc.Data, err = BlobRead(ctx, mc.BlobPath, mr.store, opts...)
	if err != nil {
		return MassifContext{}, err
	}
	mc.Tags = rr.Tags
	mc.ETag = *rr.ETag
	mc.LastRead = time.Now()
	mc.LastModfified = *rr.LastModified

	err = mc.Start.UnmarshalBinary(mc.Data)
	if err != nil {
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
func MassifIndexFromMMRIndex(massifHeight uint8, mmrIndex uint64) (uint64, error) {

	// First check if the given mmrIndex is a leaf node.
	//
	// NOTE: leaf nodes are always on height 0.
	height := mmr.IndexHeight(mmrIndex)
	if height != 0 {
		return 0, ErrNotleaf
	}

	// HeightSize returns the maximum number of nodes for a given height of MMR. Where the leaf nodes
	//  start on height 1.
	mmrSize := mmr.HeightSize(uint64(massifHeight))

	// now find the massif.
	//
	// for context, see: https://github.com/datatrails/epic-8120-scalable-proof-mechanisms/blob/main/mmr/forestrie-mmrblobs.md#blob-size
	//
	// Note: massif indexes start at 0.
	// Note: mmr indexes starts at 0.
	massifIndex := mmrIndex / mmrSize

	return massifIndex, nil

}
