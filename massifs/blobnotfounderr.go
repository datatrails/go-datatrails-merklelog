package massifs

import (
	"errors"
	"fmt"

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
