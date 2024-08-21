package massifs

import (
	"context"
	"maps"
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

func (lc *LogBlobContext) CopyTags() map[string]string {
	if lc.Tags == nil {
		return lc.Tags
	}
	m := map[string]string{}
	maps.Copy(m, lc.Tags)
	return m
}

// ReadData reads the data from the blob at BlobPath
// The various metadata fields are populated from the blob store response
// On return, the Data member containes the blob contents
func (lc *LogBlobContext) ReadData(
	ctx context.Context, store LogBlobReader, opts ...azblob.Option) error {

	var err error
	var rr *azblob.ReaderResponse

	rr, lc.Data, err = BlobRead(ctx, lc.BlobPath, store, opts...)
	if err != nil {
		return err
	}
	lc.Tags = rr.Tags

	if rr.ETag != nil {
		lc.ETag = *rr.ETag
	}

	if rr.LastModified != nil {
		lc.LastModified = *rr.LastModified
	}

	lc.LastRead = time.Now()
	lc.ContentLength = rr.ContentLength

	return nil

}
