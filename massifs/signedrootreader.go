package massifs

import (
	"context"
	"errors"

	"github.com/datatrails/go-datatrails-common/azblob"
	"github.com/datatrails/go-datatrails-common/cbor"
	"github.com/datatrails/go-datatrails-common/cose"
	"github.com/datatrails/go-datatrails-common/logger"
)

var (
	ErrLogContextNotRead = errors.New("attempted to use lastContext before it was read")
)

// SealGetter supports reading a massif seal identified by a specific massif index
type SealGetter interface {
	GetSignedRoot(
		ctx context.Context, tenantIdentity string, massifIndex uint32,
		opts ...ReaderOption,
	) (*cose.CoseSign1Message, MMRState, error)
}

type SealedState struct {
	Sign1Message cose.CoseSign1Message
	MMRState     MMRState
}

// SignedRootReader provides a context for reading the signed tree head associated with a massif
// Note: the acronym is due to RFC 9162
type SignedRootReader struct {
	log   logger.Logger
	store LogBlobReader
	codec cbor.CBORCodec
	// lastContext saves the last context read from blob store, this includes
	// Tags if they were requested
	lastContext LogBlobContext
}

func NewSignedRootReader(log logger.Logger, store LogBlobReader, codec cbor.CBORCodec) SignedRootReader {
	r := SignedRootReader{
		log:   log,
		store: store,
		codec: codec,
	}
	return r
}

// GetLastReadContext returns a copy of the most recently read context. Use this
// to get access to the tags when using WithGetTags.  If the context hasn't been
// read (directly or indirectly) an error is returned.
func (s *SignedRootReader) GetLastReadContext() (LogBlobContext, error) {
	if s.lastContext.BlobPath == "" {
		return LogBlobContext{}, ErrLogContextNotRead
	}
	return s.lastContext, nil
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
	s.lastContext = logBlobContext
	return logBlobContext, count, nil
}

func (s *SignedRootReader) ReadLogicalContext(
	ctx context.Context, logContext LogBlobContext,
	opts ...azblob.Option,
) (*cose.CoseSign1Message, MMRState, error) {

	err := logContext.ReadData(ctx, s.store, opts...)
	if err != nil {
		return nil, MMRState{}, err
	}
	s.lastContext = logContext

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
) (*cose.CoseSign1Message, MMRState, uint64, error) {

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
	opts ...ReaderOption,
) (*cose.CoseSign1Message, MMRState, error) {

	options := ReaderOptions{}
	for _, o := range opts {
		o(&options)
	}

	blobPath := TenantMassifSignedRootPath(tenantIdentity, massifIndex)

	logContext := LogBlobContext{
		BlobPath: blobPath,
	}

	signed, unverifiedState, err := s.ReadLogicalContext(ctx, logContext, options.remoteReadOpts...)

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
) (*cose.CoseSign1Message, MMRState, error) {

	logContext := LogBlobContext{
		BlobPath: TenantMassifSignedRootPath(tenantIdentity, massifIndex),
	}
	err := logContext.ReadData(ctx, s.store, opts...)
	if err != nil {
		return nil, MMRState{}, err
	}
	s.lastContext = logContext
	signed, unverifiedState, err := DecodeSignedRoot(s.codec, logContext.Data)
	if err != nil {
		return nil, MMRState{}, err
	}

	return signed, unverifiedState, err
}
