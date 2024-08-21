package massifs

import (
	"crypto/ecdsa"

	"github.com/datatrails/go-datatrails-common/azblob"
	"github.com/datatrails/go-datatrails-common/cbor"
)

// ReaderOptions provides options for MassifReader and SignedRootReader
// implementations implementations are expected to simply ignore options that
// they don't support
type ReaderOptions struct {
	noGetRootSupport bool

	massifHeight uint8

	// The following options are only relevant to reader implementations that interact with the blobs api.

	// options that are forwarded when issuing a read blob call
	remoteReadOpts []azblob.Option
	// options that are forwarded when issuing a list blobs call
	remoteListOpts []azblob.Option

	// options that are forwarded when issuing a filter blobs call
	remoteFilterOpts []azblob.Option

	// The following options are only relevant when the reader is configured to read seals

	sealGetter SealGetter
	codec      *cbor.CBORCodec

	// Used by methods which support verifying consistency against a state from
	// an independent trusted source.
	trustedBaseState    *MMRState
	trustedSealerPubKey *ecdsa.PublicKey
}

// ReaderOptionsCopy creates an independent of the opts
func ReaderOptionsCopy(opts ReaderOptions) ReaderOptions {
	cpy := opts

	cpy.remoteReadOpts = make([]azblob.Option, len(opts.remoteReadOpts))
	copy(cpy.remoteReadOpts, opts.remoteReadOpts)

	cpy.remoteListOpts = make([]azblob.Option, len(opts.remoteListOpts))
	copy(cpy.remoteListOpts, opts.remoteListOpts)

	cpy.remoteFilterOpts = make([]azblob.Option, len(opts.remoteFilterOpts))
	copy(cpy.remoteFilterOpts, opts.remoteFilterOpts)
	return cpy
}

// NewReaderOptions creates a new ReaderOptions object with the provided options
// Typically, this is used for mocking as the options values are private
func NewReaderOptions(baseOpts ReaderOptions, opts ...ReaderOption) ReaderOptions {
	options := ReaderOptionsCopy(baseOpts)
	for _, o := range opts {
		o(&options)
	}
	return options
}

type ReaderOption func(*ReaderOptions)

func WithSealGetter(getter SealGetter) ReaderOption {
	return func(opts *ReaderOptions) {
		opts.sealGetter = getter
	}
}

// WithTrustedBaseState can be used with methods which verify log consistency,
// to ensure the log is consistent with an independently trusted copy of a
// previous log sate.
func WithTrustedBaseState(state MMRState) ReaderOption {
	return func(opts *ReaderOptions) {
		opts.trustedBaseState = &state
	}
}

func WithTrustedSealerPub(pub *ecdsa.PublicKey) ReaderOption {
	return func(opts *ReaderOptions) {
		opts.trustedSealerPubKey = pub
	}
}

// WithoutGetRootSupport disables the random access map for the peak stack.
// This typically should only be set by log builders
func WithoutGetRootSupport() ReaderOption {
	return func(opts *ReaderOptions) {
		opts.noGetRootSupport = true
	}
}

func WithMassifHeight(massifHeight uint8) ReaderOption {
	return func(opts *ReaderOptions) {
		opts.massifHeight = massifHeight
	}
}

func WithReadBlobOption(opt azblob.Option) ReaderOption {
	return func(opts *ReaderOptions) {
		opts.remoteReadOpts = append(opts.remoteReadOpts, opt)
	}
}

func WithListBlobOption(opt azblob.Option) ReaderOption {
	return func(opts *ReaderOptions) {
		opts.remoteListOpts = append(opts.remoteListOpts, opt)
	}
}

func WithFilterBlobsOption(opt azblob.Option) ReaderOption {
	return func(opts *ReaderOptions) {
		opts.remoteFilterOpts = append(opts.remoteFilterOpts, opt)
	}
}

func WithCBORCodec(codec cbor.CBORCodec) ReaderOption {
	return func(o *ReaderOptions) {
		o.codec = &codec
	}
}
