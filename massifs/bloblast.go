package massifs

import (
	"context"
	"errors"

	"github.com/datatrails/go-datatrails-common/azblob"
)

var (
	ErrBlobNotFound = errors.New("the blob was not found")
)

// LastPrefixedBlob returns the details of last blob found under the prefix path
// And the total number of blobs under the path.
func LastPrefixedBlob(
	ctx context.Context, store LogBlobReader, blobPrefixPath string,
	opts ...azblob.Option,
) (LogBlobContext, uint64, error) {

	bc := LogBlobContext{}

	var foundCount uint64

	opts = append([]azblob.Option{azblob.WithListPrefix(blobPrefixPath)}, opts...)

	var marker azblob.ListMarker
	for {
		r, err := store.List(ctx, append(opts, azblob.WithListMarker(marker))...)
		if err != nil {
			return bc, foundCount, err
		}
		if len(r.Items) == 0 {
			return bc, foundCount, nil
		}

		foundCount += uint64(len(r.Items))

		// we want the _last_ listed, so we just keep over-writing
		i := r.Items[len(r.Items)-1]
		bc.ETag = *i.Properties.Etag
		bc.LastModified = *i.Properties.LastModified
		if i.Properties.ContentLength != nil {
			bc.ContentLength = *i.Properties.ContentLength
		}
		bc.BlobPath = *i.Name
		marker = r.Marker
		if marker == nil || *marker == "" {
			bc.Tags = listResponseTags(i.BlobTags)
			break
		}
	}

	return bc, foundCount, nil
}

// FirstPrefixedBlob returns the first blob found under the prefix path
func FirstPrefixedBlob(
	ctx context.Context, store LogBlobReader, blobPrefixPath string,
	opts ...azblob.Option,
) (LogBlobContext, error) {

	bc := LogBlobContext{}

	opts = append([]azblob.Option{
		azblob.WithListPrefix(blobPrefixPath),
		azblob.WithListMaxResults(1),
	}, opts...)

	r, err := store.List(ctx, opts...)
	if err != nil {
		return bc, err
	}
	if len(r.Items) == 0 {
		return bc, ErrBlobNotFound
	}

	i := r.Items[0]
	bc.ETag = *i.Properties.Etag
	bc.LastModified = *i.Properties.LastModified
	if i.Properties.ContentLength != nil {
		bc.ContentLength = *i.Properties.ContentLength
	}
	bc.BlobPath = *i.Name
	bc.Tags = listResponseTags(r.Items[0].BlobTags)

	return bc, nil
}

// PrefixedBlobLastN returns contexts for the last n blobs under the provided prefix.
//
// The number of items in the returned tail is always min(massifCount, n)
// Un filled items are zero valued.
func PrefixedBlobLastN(
	ctx context.Context,
	store LogBlobReader,
	blobPrefixPath string,
	n int,
	opts ...azblob.Option,
) ([]LogBlobContext, uint64, error) {

	opts = append([]azblob.Option{azblob.WithListPrefix(blobPrefixPath)}, opts...)

	tail := make([]LogBlobContext, n)

	var foundCount uint64

	var marker azblob.ListMarker
	for {
		r, err := store.List(ctx, append(opts, azblob.WithListMarker(marker))...)
		if err != nil {
			return tail, foundCount, err
		}
		if len(r.Items) == 0 {
			return tail, foundCount, nil
		}

		foundCount += uint64(len(r.Items))

		// The stale items are those from the previous round that can be
		// replaced by the current. Typically, len(r.Items) will be greater than
		// n and so it will be n. Note that stale is > 0 here due to the len 0
		// check above.
		stale := min(len(r.Items), n)

		// copy the items *after* the stale items to the front.
		if stale != n {
			copy(tail, tail[n-stale-1:])
		}

		for i := 0; i < stale; i++ {

			// stale is also the count of items we are taking from items.

			it := r.Items[len(r.Items)-stale+i]
			tail[n-stale+i].ETag = *it.Properties.Etag
			tail[n-stale+i].LastModified = *it.Properties.LastModified
			if it.Properties.ContentLength != nil {
				tail[n-stale+i].ContentLength = *it.Properties.ContentLength
			}
			tail[n-stale+i].BlobPath = *it.Name
			tail[n-stale+i].Tags = listResponseTags(it.BlobTags)
		}

		marker = r.Marker
		if marker == nil || *marker == "" {
			break
		}
	}

	// Note massifIndex will be zero, the id of the first massif blob
	return tail, foundCount, nil
}
