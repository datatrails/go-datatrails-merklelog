package mmrblobs

import (
	"context"
	"errors"
	"io"

	"github.com/datatrails/go-datatrails-common/azblob"
)

// BlobRead reads the blob of the given store.
func BlobRead(
	ctx context.Context, blobPath string, store massifReader,
	opts ...azblob.Option) (*azblob.ReaderResponse, []byte, error) {

	rr, err := store.Reader(ctx, blobPath, opts...)
	if err != nil {
		return nil, nil, err
	}

	data := make([]byte, rr.ContentLength)
	read := int64(0)
	for read < rr.ContentLength {
		n, err := rr.Reader.Read(data[read:])
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, nil, err
		}
		read += int64(n)
	}

	// The reader is now definitely exhausted for the purpose it was created. To
	// avoid odd effects, or accidental misuse we nill it out. And we do so regardless of error.

	rr.Reader = nil // The caller has no use for this

	// If we read less. truncate the buffer
	if read < int64(len(data)) {
		data = data[0:read]
	}
	return rr, data, nil
}
