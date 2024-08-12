package massifs

import (
	"github.com/datatrails/go-datatrails-common/azblob"
	"github.com/datatrails/go-datatrails-common/cbor"
)

// ReaderOptions provides options for MassifReader and SignedRootReader
// implementations implementations are expected to simply ignore options that
// they don't support
type ReaderOptions struct {
	noGetRootSupport bool

	requireMassifHeight uint8

	// The following options are only relevant to reader implementations that interact with the blobs api.

	// options that are forwarded when issuing a read blob call
	remoteReadOpts []azblob.Option
	// options that are forwarded when issuing a list blobs call
	remoteListOpts []azblob.Option

	// options that are forwarded when issuing a filter blobs call
	remoteFilterOpts []azblob.Option

	// The following options are only relevant when the reader is configured to read seals
	codec cbor.CBORCodec
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

type ReaderOption func(*ReaderOptions)

// WithoutGetRootSupport disables the random access map for the peak stack.
// This typically should only be set by log builders
func WithoutGetRootSupport() ReaderOption {
	return func(opts *ReaderOptions) {
		opts.noGetRootSupport = true
	}
}

func WithRequireMassifHeight(massifHeight uint8) ReaderOption {
	return func(opts *ReaderOptions) {
		opts.requireMassifHeight = massifHeight
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
		o.codec = codec
	}
}
