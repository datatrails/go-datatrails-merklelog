package massifs

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azStorageBlob "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

const (
	azblobBlobNotFound = "BlobNotFound"
)

func AsStorageError(err error) (azStorageBlob.StorageError, bool) {
	serr := &azStorageBlob.StorageError{}
	//nolint
	ierr, ok := err.(*azStorageBlob.InternalError)
	if ierr == nil || !ok {
		return azStorageBlob.StorageError{}, false
	}
	if !ierr.As(&serr) {
		return azStorageBlob.StorageError{}, false
	}
	return *serr, true
}

func AsResponseError(err error) (azcore.ResponseError, bool) {

	var ok bool
	var rerr *azcore.ResponseError

	if rerr, ok = err.(*azcore.ResponseError); ok {
		return *rerr, true
	}

	// check for an InternalError that has ResponseError as its cause
	rerr = &azcore.ResponseError{}

	//nolint
	ierr, ok := err.(*azStorageBlob.InternalError)
	if ierr == nil || !ok {
		return azcore.ResponseError{}, false
	}
	if !ierr.As(&rerr) {
		return azcore.ResponseError{}, false
	}
	return *rerr, true
}

// WrapBlobNotFound tranlsates the err to ErrBlobNotFound if the actual error is
// the azure sdk blob not found error. In all cases where the original err is
// not the azure BlobNot found, the original err is returned as is. Including
// the case where it is nil
func WrapBlobNotFound(err error) error {
	if err == nil {
		return nil
	}
	serr, ok := AsStorageError(err)
	if !ok {
		return err
	}
	if serr.ErrorCode != azblobBlobNotFound {
		return err
	}
	return fmt.Errorf("%s: %w", err.Error(), ErrBlobNotFound)
}

func IsBlobNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrBlobNotFound) {
		return true
	}
	serr, ok := AsStorageError(err)
	if !ok {
		return false
	}
	if serr.ErrorCode != azblobBlobNotFound {
		return false
	}
	return true
}

// IsRateLimiting detects if the error is HTTP Status 429 Too Many Requests
// The recomended wait time is returned if it is available. If the returned wait
// time is zero, the caller should apply an appropriate default backoff.
func IsRateLimiting(err error) (time.Duration, bool) {
	if err == nil {
		return 0, false
	}
	rerr, ok := AsResponseError(err)
	if !ok {
		return 0, false
	}
	if rerr.StatusCode != http.StatusTooManyRequests {
		return 0, false
	}

	// It is a 429, check if there is a Retry-After header and return the indicated time if possible.

	// Retry-After header is optional, the
	if rerr.RawResponse == nil {
		return 0, true
	}
	retryAfter := rerr.RawResponse.Header.Get("Retry-After")
	if retryAfter == "" {
		return 0, true
	}

	// Try to parse Retry-After as an integer (seconds)
	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		return time.Duration(seconds) * time.Second, true
	}

	// Try to parse Retry-After as a date
	if retryTime, err := http.ParseTime(retryAfter); err == nil {
		retryTime = retryTime.In(time.UTC) // crucial, as Until does not work with different locations
		return time.Until(retryTime), true
	}

	// couldn't parse the time, but it is definitely a 429. the caller should apply an appropriate default backoff.
	return 0, true
}
