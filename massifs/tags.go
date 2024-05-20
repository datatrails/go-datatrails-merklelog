package massifs

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
)

// Note: we favour hex encoding because hex values sort lexically and blob tags
// only support lexical sorts

var (
	ErrHex64TagOverflow       = errors.New("a tag value expected to be 64 bit hex had more than 8 bytes of data")
	ErrMissingFirstIndexTag   = errors.New("the required tag 'firstindex' is missing")
	ErrMissingLastIDTag       = errors.New("the required tag 'lastid' is missing")
	ErrIncorrectFirstIndexTag = errors.New("the required tag 'firstindex' is present but the value doesn't match the log")
)

const (
	TagKeyFirstIndex = "firstindex"
	TagKeyLastID     = "lastid"
	TagFirstIndexFmt = "%016x"
)

func GetFirstIndex(tags map[string]string) (uint64, error) {
	firstIndexTag, ok := tags[TagKeyFirstIndex]
	if !ok {
		return 0, ErrMissingFirstIndexTag
	}

	return DecodeTagHex64(firstIndexTag)
}

func GetLastIDHex(tags map[string]string) string {
	lastIDTag, ok := tags[TagKeyLastID]
	if !ok {
		return ""
	}
	return lastIDTag
}

func SetFirstIndex(firstIndex uint64, tags map[string]string) {
	// take care to 0 pad to preserve lexical sort
	tags[TagKeyFirstIndex] = fmt.Sprintf(TagFirstIndexFmt, firstIndex)
}

// DecodeHex64 decodes a tag value which is a big endian hexidecimal string
func DecodeTagHex64(tagValue string) (uint64, error) {
	b, err := hex.DecodeString(tagValue)
	if err != nil {
		return 0, err
	}
	if len(b) > 8 {
		return 0, fmt.Errorf("%w: %s", ErrHex64TagOverflow, tagValue)
	}
	return binary.BigEndian.Uint64(b), nil
}

func EncodeTagHex64(tagValue uint64) string {
	return fmt.Sprintf("%016x", tagValue)
}
