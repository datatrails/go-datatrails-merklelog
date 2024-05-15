package massifs

import (
	"context"

	"github.com/datatrails/go-datatrails-common/azblob"
	"github.com/datatrails/go-datatrails-common/cbor"
	dtcose "github.com/datatrails/go-datatrails-common/cose"
	"github.com/datatrails/go-datatrails-common/logger"
)

// SignedRootReader provides a context for reading the signed tree head associated with a massif
// Note: the acronym is due to RFC 9162
type SignedRootReader struct {
	log   logger.Logger
	store logBlobReader
	codec cbor.CBORCodec
}

func NewSignedRootReader(log logger.Logger, store logBlobReader, codec cbor.CBORCodec) SignedRootReader {
	r := SignedRootReader{
		log:   log,
		store: store,
		codec: codec,
	}
	return r
}

func (s *SignedRootReader) GetLazyContext(
	ctx context.Context, tenantIdentity string, which LogicalBlob,
	opts ...azblob.Option,
) (LogBlobContext, uint64, error) {
	blobPrefixPath := TenantMassifSignedRootsPrefix(tenantIdentity)

	var count uint64
	var err error
	var logBlobContext LogBlobContext
	switch which {
	case FirstBlob:
		logBlobContext, err = FirstPrefixedBlob(ctx, s.store, blobPrefixPath, opts...)
	case LastBlob:
		logBlobContext, count, err = LastPrefixedBlob(ctx, s.store, blobPrefixPath, opts...)
		// force an error for no blobs found, but respect the original err if there was one.
		if err == nil && count == 0 {
			return LogBlobContext{}, count, ErrBlobNotFound
		}
	}
	if err != nil {
		return LogBlobContext{}, 0, err
	}
	return logBlobContext, count, nil
}

func (s *SignedRootReader) ReadLogicalContext(
	ctx context.Context, logContext LogBlobContext,
	opts ...azblob.Option,
) (*dtcose.CoseSign1Message, MMRState, error) {

	err := logContext.ReadData(ctx, s.store, opts...)
	if err != nil {
		return nil, MMRState{}, err
	}

	signed, unverifiedState, err := DecodeSignedRoot(s.codec, logContext.Data)
	if err != nil {
		return nil, MMRState{}, err
	}

	return signed, unverifiedState, err
}

// Note that the log head can be arbitrarily ahead of the root signatures.
//
// When confirming the log we must verify the consistency against the previously
// signed root. In the case where we are signing the first root for a massif,
// the previous root will be for the previous massif.
func (s *SignedRootReader) GetLatestSignedRoot(
	ctx context.Context, tenantIdentity string,
	opts ...azblob.Option,
) (*dtcose.CoseSign1Message, MMRState, uint64, error) {

	blobPrefixPath := TenantMassifSignedRootsPrefix(tenantIdentity)

	// GetLazyContext allows tags for the list operation, so use that and then use ReadData if you really need different opts for each.
	logContext, count, err := LastPrefixedBlob(ctx, s.store, blobPrefixPath)
	if err != nil {
		return nil, MMRState{}, 0, err
	}
	if count == 0 {
		return nil, MMRState{}, 0, nil
	}
	signed, unverifiedState, err := s.ReadLogicalContext(ctx, logContext, opts...)

	return signed, unverifiedState, count, err
}

// GetSignedRoot gets the signed root for the massif at the given massifIndex.
//
// When confirming the log we must verify the consistency against the previously
// signed root. In the case where we are signing the first root for a massif,
// the previous root will be for the previous massif.
func (s *SignedRootReader) GetSignedRoot(
	ctx context.Context, tenantIdentity string, massifIndex uint32,
	opts ...azblob.Option,
) (*dtcose.CoseSign1Message, MMRState, error) {

	blobPath := TenantMassifSignedRootPath(tenantIdentity, massifIndex)

	logContext := LogBlobContext{
		BlobPath: blobPath,
	}

	signed, unverifiedState, err := s.ReadLogicalContext(ctx, logContext, opts...)

	return signed, unverifiedState, err
}

// Get the signed tree head (SignedRoot) for the mmr massif.
//
// NOTICE: TO VERIFY YOU MUST obtain the mmr root from the log using the
// MMRState.MMRSize in the returned MMRState. See {@link VerifySignedRoot}
//
// This may not be the latest mmr head, but it will be the latest for the
// argument massifIndex. If the identified massif is complete, the returned SignedRoot
// will remain valid for the lifetime of the mmr. Due to the 'asynchronous'
// property of mmrs and 'old-accumulator compatibility', see
// {@link // https://eprint.iacr.org/2015/718.pdf}
func (s *SignedRootReader) GetLatestMassifSignedRoot(
	ctx context.Context, tenantIdentity string, massifIndex uint32,
	opts ...azblob.Option,
) (*dtcose.CoseSign1Message, MMRState, error) {

	lc := LogBlobContext{
		BlobPath: TenantMassifSignedRootPath(tenantIdentity, massifIndex),
	}
	err := lc.ReadData(ctx, s.store, opts...)
	if err != nil {
		return nil, MMRState{}, err
	}
	signed, unverifiedState, err := DecodeSignedRoot(s.codec, lc.Data)
	if err != nil {
		return nil, MMRState{}, err
	}

	return signed, unverifiedState, err
}
