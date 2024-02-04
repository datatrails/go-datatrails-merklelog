package mmrblobs

import (
	"context"

	"github.com/datatrails/go-datatrails-common/azblob"
)

type logBlobReader interface {
	Reader(
		ctx context.Context,
		identity string,
		opts ...azblob.Option,
	) (*azblob.ReaderResponse, error)

	List(ctx context.Context, opts ...azblob.Option) (*azblob.ListerResponse, error)
}
