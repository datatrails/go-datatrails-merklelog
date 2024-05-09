package massifs

import (
	"context"
	"time"

	"github.com/datatrails/go-datatrails-common/azblob"
)

// LogBlobContext provides a common context for reading & writing log blobs
//
// The log is comprised of a series of numbered blobs. With one blob per
// 'massif'. There are a few different types, and each type, due to how blob
// listing works, is stored under a distinct prefix. All operations at the head
// of the log, regardless of the specicic blob type, need a method to find the
// last (most recently created) blob under a prefix
type LogBlobContext struct {
	BlobPath      string
	ETag          string
	Tags          map[string]string
	LastRead      time.Time
	LastModified  time.Time
	Data          []byte
	ContentLength int64
}

// ReadData reads the data from the blob at BlobPath
// The various metadata fields are populated from the blob store response
// On return, the Data member containes the blob contents
func (lc *LogBlobContext) ReadData(
	ctx context.Context, store logBlobReader, opts ...azblob.Option) error {

	var err error
	var rr *azblob.ReaderResponse

	rr, lc.Data, err = BlobRead(ctx, lc.BlobPath, store, opts...)
	if err != nil {
		return err
	}
	lc.Tags = rr.Tags
	if lc.Tags != nil {
		if rr.Tags != nil {

		}
	}
	lc.ETag = *rr.ETag
	lc.LastRead = time.Now()
	lc.LastModified = *rr.LastModified
	lc.ContentLength = rr.ContentLength

	return nil

}
