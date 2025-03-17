package massifs

import (
	"context"

	azStorageBlob "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/datatrails/go-datatrails-common/azblob"
)

type VisitFilterResponse func(ctx context.Context, store LogBlobReader, it *azStorageBlob.FilterBlobItem) (bool, error)

// FilterBlobs selects blobs using the provided filter expression
// and application of the provided visitor (which may be nil)
// tagsFilter examples:
//
//	All blobs in a storage account
//		"cat='tiger' AND penguin='emperorpenguin'"
//	All blobs in a specific container
//		"@container='zoo' AND cat='tiger' AND penguin='emperorpenguin'"
func FilterBlobs(
	ctx context.Context, store LogBlobReader,
	tagsFilter string,
	visit VisitFilterResponse,
	marker azblob.ListMarker,
	opts ...azblob.Option,
) ([]*azStorageBlob.FilterBlobItem, azblob.ListMarker, error) {

	opts = append(opts, []azblob.Option{azblob.WithListMarker(marker)}...)
	r, err := store.FilteredList(ctx, tagsFilter, opts...)
	if err != nil {
		// It is important to return the same marker in the event of an error.
		// Only the caller can usefully decide if to retry the page or the whole
		// lot.
		return nil, marker, err
	}

	var newFound []*azStorageBlob.FilterBlobItem

	// If there is no visitor, just return all the found items
	if visit == nil {
		newFound = append(newFound, r.Items...)

		return newFound, r.Marker, nil
	}

	// Ok, there is a visitor. Only return those items for which it returns ok=true
	for i := range r.Items {
		ok, err := visit(ctx, store, r.Items[i])
		if err != nil {
			// Note that we return the original marker, if the caller wants to
			// re-try the page they can
			return nil, marker, err
		}
		if !ok {
			continue
		}
		newFound = append(newFound, r.Items[i])
	}

	return newFound, r.Marker, nil
}

// ParseIdentifyingSegment extracts a uniquely identifying string from the blobPath for the benefit of EnumerateIdentifierPaths
type ParseIdentifyingSegment func(blobPath string) (string, error)

// EnumerateIdentifiedPaths finds the unique sub paths of blobPrefixPath The
// ParseIdentifyingSegment call back finds the uniquely identifying key in the
// sub path and returns it, and the enumeration accumulates the unique set of
// those keys. If the parse callback errors, the enumeration is terminated and
// that error is returned.
func EnumerateIdentifiedPaths(
	ctx context.Context, store LogBlobReader, blobPrefixPath string,
	parseID ParseIdentifyingSegment,
	found map[string]any,
	marker azblob.ListMarker,
	opts ...azblob.Option,
) ([]string, azblob.ListMarker, error) {

	opts = append(opts, []azblob.Option{azblob.WithListPrefix(blobPrefixPath), azblob.WithListMarker(marker)}...)

	var newFound []string

	r, err := store.List(ctx, opts...)
	if err != nil {
		// It is important to return the same marker in the event of an error.
		// Only the caller can usefully decide if to retry the page or the whole
		// lot.
		return nil, marker, err
	}

	for i := range r.Items {
		id, err := parseID(*r.Items[i].Name)
		if err != nil {
			// This is a situation where the paths to the blobs are invalid.
			// Crashing out is the only reasonable response. So we don't
			// preserve the marker.
			return nil, nil, err
		}
		if _, ok := found[id]; ok {
			continue
		}
		newFound = append(newFound, id)
		found[id] = true
	}

	return newFound, r.Marker, nil
}
