package massifs

import (
	"context"
	"io"

	"github.com/datatrails/go-datatrails-common/azblob"
)

type massifStore interface {
	Put(
		ctx context.Context,
		identity string,
		source io.ReadSeekCloser,
		opts ...azblob.Option,
	) (*azblob.WriteResponse, error)
	Reader(
		ctx context.Context,
		identity string,
		opts ...azblob.Option,
	) (*azblob.ReaderResponse, error)

	FilteredList(ctx context.Context, tagsFilter string, opts ...azblob.Option) (*azblob.FilterResponse, error)
	List(ctx context.Context, opts ...azblob.Option) (*azblob.ListerResponse, error)
}
