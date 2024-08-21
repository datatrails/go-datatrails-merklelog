package massifs

import (
	"context"

	"github.com/datatrails/go-datatrails-common/azblob"
)

type LogBlobReader interface {
	Reader(
		ctx context.Context,
		identity string,
		opts ...azblob.Option,
	) (*azblob.ReaderResponse, error)

	FilteredList(ctx context.Context, tagsFilter string, opts ...azblob.Option) (*azblob.FilterResponse, error)
	List(ctx context.Context, opts ...azblob.Option) (*azblob.ListerResponse, error)
}
