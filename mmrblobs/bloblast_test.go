package mmrblobs

import (
	"context"
	"fmt"
	"testing"
	"time"

	azStorageBlob "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/datatrails/go-datatrails-common/azblob"
	"github.com/stretchr/testify/assert"
)

func TestLastPrefixedBlob(t *testing.T) {
	type args struct {
		store logBlobReader
	}
	tests := []struct {
		name        string
		args        args
		want        LogBlobContext
		massifCount uint64
		wantBlob    string
		wantErr     bool
	}{
		{
			name:        "no blobs",
			args:        args{newLastNBlobStore()},
			massifCount: 0,
		},
		{
			name:        "one blob",
			args:        args{newLastNBlobStore(1)},
			massifCount: 1,
			wantBlob:    "blob-0",
		},
		{
			name:        "two blobs",
			args:        args{newLastNBlobStore(1, 1)},
			massifCount: 2,
			wantBlob:    "blob-1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := LastPrefixedBlob(
				context.Background(),
				tt.args.store,
				"prefix/path/")
			if (err != nil) != tt.wantErr {
				t.Errorf("LastPrefixedBlob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got1 != tt.massifCount {
				t.Errorf("LastPrefixedBlob() got1 = %v, want %v", got1, tt.massifCount)
			}
			if tt.wantBlob != "" {
				assert.Equal(t, tt.wantBlob, got.BlobPath)
			}
		})
	}
}

func TestPrefixedBlobLastN(t *testing.T) {

	type args struct {
		store logBlobReader
		n     int
	}
	tests := []struct {
		name        string
		args        args
		massifCount uint64
		wantBlobs   []string
		wantErr     bool
	}{

		{
			name:        "fewer items than n",
			args:        args{newLastNBlobStore(1, 1), 3},
			massifCount: 1 + 1,
			wantBlobs:   []string{"", "blob-0", "blob-1"},
		},

		{
			name:        "single item per request",
			args:        args{newLastNBlobStore(1, 1, 1, 1, 1), 3},
			massifCount: 1 + 1 + 1 + 1 + 1,
			wantBlobs:   []string{"blob-2", "blob-3", "blob-4"},
		},
		{
			name:        "fewer items per request than n",
			args:        args{newLastNBlobStore(1, 2, 1, 2, 1), 3},
			massifCount: 1 + 2 + 1 + 2 + 1,
			wantBlobs:   []string{"blob-4", "blob-5", "blob-6"},
		},
		{
			name:        "fewer and greater items per request than n",
			args:        args{newLastNBlobStore(1, 2, 5, 2, 1), 3},
			massifCount: 1 + 2 + 5 + 2 + 1,
			wantBlobs:   []string{"blob-8", "blob-9", "blob-10"},
		},

		{
			name:        "more items than tail len",
			args:        args{newLastNBlobStore(5, 5, 5), 3},
			massifCount: 5 + 5 + 5,
			wantBlobs:   []string{"blob-12", "blob-13", "blob-14"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, massifCount, err := PrefixedBlobLastN(
				context.Background(),
				tt.args.store,
				"prefix/path/",
				tt.args.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("PrefixedBlobLastN() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if massifCount != tt.massifCount {
				t.Errorf("PrefixedBlobLastN() got1 = %v, want %v", massifCount, tt.massifCount)
			}
			if tt.wantBlobs != nil {
				for i, want := range tt.wantBlobs {
					assert.Equal(t, want, got[i].BlobPath)
				}
			}

			// Check the number of tail items which are populated is correct.
			// Note that when the tail is not full, the empty items are at the
			// *front* So, if n is 3, but there are only two blobs the tail
			// population looks like this.
			// [empty, blob-last-1-info, blob-last-info]

			expectedTailLen := min(massifCount, uint64(tt.args.n))
			gotTailLen := uint64(0)
			for i := uint64(tt.args.n) - expectedTailLen; i < uint64(len(got)); i++ {
				if len(got[i].BlobPath) != 0 {
					gotTailLen++
				}
			}
			assert.Equal(t, expectedTailLen, gotTailLen)
		})
	}
}

func newLastNBlobStore(itemCounts ...int) logBlobReader {

	series := newmockblobseries(itemCounts...)
	return &mockLastNBlobStore{
		series: series,
	}
}

type properties struct {
	Name          *string
	ETag          *string
	LastModfified *time.Time
}

func newprops(name string, etag string, t time.Time) properties {
	p := properties{
		Name:          new(string),
		ETag:          new(string),
		LastModfified: new(time.Time),
	}
	*p.Name = name
	*p.ETag = etag
	*p.LastModfified = t
	return p
}

func newmockblobseries(itemCounts ...int) [][]properties {

	var series [][]properties
	var iFirstRequestBlob int
	for _, itemCount := range itemCounts {
		items := make([]properties, itemCount)
		for i := 0; i < itemCount; i++ {
			items[i] = newprops(
				fmt.Sprintf("blob-%d", iFirstRequestBlob+i),
				fmt.Sprintf("tag-%d", iFirstRequestBlob+i),
				time.Now(),
			)
		}
		series = append(series, items)
		iFirstRequestBlob += itemCount
	}
	return series
}

type mockLastNBlobStore struct {
	series    [][]properties
	callCount int
}

func (s *mockLastNBlobStore) Reader(
	ctx context.Context,
	identity string,
	opts ...azblob.Option,
) (*azblob.ReaderResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *mockLastNBlobStore) List(ctx context.Context, opts ...azblob.Option) (*azblob.ListerResponse, error) {
	s.callCount += 1
	if s.callCount == len(s.series)+1 {
		// returning an empty response is the terminator, but we must only do so once
		return &azblob.ListerResponse{}, nil
	}
	if s.callCount > len(s.series)+1 {
		return nil, fmt.Errorf("no more test data")
	}

	blobs := s.series[s.callCount-1]

	items := make([]*azStorageBlob.BlobItemInternal, len(blobs))
	for i := 0; i < len(blobs); i++ {
		items[i] = &azStorageBlob.BlobItemInternal{
			Name:       blobs[i].Name,
			Properties: &azStorageBlob.BlobPropertiesInternal{},
		}
		items[i].Properties.Etag = blobs[i].ETag
		items[i].Properties.LastModified = blobs[i].LastModfified
	}
	return &azblob.ListerResponse{
		Marker: new(string),
		Items:  items,
	}, nil
}
