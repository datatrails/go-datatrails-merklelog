package watcher

import (
	"math/rand"
	"slices"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/datatrails/go-datatrails-merklelog/massifs"
)

// LogTail records the newest (highest numbered) massif path in a log It is used
// to represent both the most recent massif log blob, and the most recent massif
// seal blob
type LogTail struct {
	Tenant string
	Path   string
	Number uint32
	Ext    string
	LastID string
}

// LogTailCollator is used to collate the most recently modified massif blob paths for all tenants in a given time horizon
type LogTailCollator struct {
	Massifs map[string]LogTail
	Seals   map[string]LogTail
}

// NewLogTail parses the log tail information from path and returns a LogTail
func NewLogTail(path string) (LogTail, error) {
	number, ext, err := massifs.ParseMassifPathNumberExt(path)
	if err != nil {
		return LogTail{}, err
	}
	tenant, err := massifs.ParseMassifPathTenant(path)
	if err != nil {
		return LogTail{}, err
	}

	return LogTail{
		Tenant: tenant,
		Path:   path,
		Number: number,
		Ext:    ext,
		LastID: "",
	}, nil
}

// TryReplacePath considers if the other path is more recent. If it is, it
// replaces the values on the current tail with those parsed from other and
// returns true.  Returns false if other is older than the tail.
func (l *LogTail) TryReplacePath(path string) bool {

	if l.Ext == massifs.V1MMRMassifExt && !massifs.IsMassifPathLike(path) {
		return false
	}

	if l.Ext == massifs.V1MMRSealSignedRootExt && !massifs.IsSealPathLike(path) {
		return false
	}

	number, ext, err := massifs.ParseMassifPathNumberExt(path)
	if err != nil {
		return false
	}
	if number <= l.Number {
		return false
	}
	l.Number = number
	l.Ext = ext
	return true
}

// TryReplaceTail considers if the other tail is more recent If it is, it
// replaces the values on the current tail with those copied from other and
// returns true.  Returns false if other is older than the tail.
func (l *LogTail) TryReplaceTail(other LogTail) bool {

	// The replacement needs to be for the other log
	if l.Tenant != other.Tenant || l.Ext != other.Ext {
		return false
	}

	if other.Number <= l.Number {
		return false
	}
	l.Path = other.Path
	l.Number = other.Number
	l.Ext = other.Ext

	return true
}

// NewLogTailCollator creates a log tail collator
func NewLogTailCollator() LogTailCollator {
	return LogTailCollator{
		Massifs: make(map[string]LogTail),
		Seals:   make(map[string]LogTail),
	}
}

// sortMapOfLogTails returns a lexically sorted list of the keys to map of
// LogTails It's not a stable sort, keys that are in the right place to start
// with may move as a result of this call.
func sortMapOfLogTails(m map[string]LogTail) []string {
	// The go lang community seems pretty divided on O(1)iterators, and I think this is still "the way"
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// shuffleMapOfLogTails returns the list of keys shuffled using rand.Shuffle
// This should be used to avoid odd biases due to fixed order treatment of tenants.
func shuffleMapOfLogTails(m map[string]LogTail) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	rand.Shuffle(len(keys), func(i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
	})
	return keys
}

// MassifTenants returns the keys of the massifs map specifically shuffled to
// avoid biasing service based on lexical order of tenant identities or go lang
// default key ordering
func (c LogTailCollator) MassifTenants() []string {
	return shuffleMapOfLogTails(c.Massifs)
}

// SortedMassifTenants returns the keys of the massifs map in sorted order
func (c LogTailCollator) SortedMassifTenants() []string {
	return sortMapOfLogTails(c.Massifs)
}

// SealedTenants returns the keys of the seals map specifically shuffled to
// avoid biasing service based on lexical order of tenant identities or go lang
// default key ordering
func (c LogTailCollator) SealedTenants() []string {
	return shuffleMapOfLogTails(c.Seals)
}

// SortedSealedTenants returns the keys of the massifs map in sorted order
func (c LogTailCollator) SortedSealedTenants() []string {
	return sortMapOfLogTails(c.Seals)
}

func collectTags(aztags *azblob.BlobTags) map[string]string {
	if aztags == nil || len(aztags.BlobTagSet) == 0 {
		return map[string]string{}
	}
	tags := map[string]string{}
	for _, azTag := range aztags.BlobTagSet {
		if azTag.Key == nil || azTag.Value == nil {
			continue
		}
		tags[*azTag.Key] = *azTag.Value
	}
	return tags
}

// collectPageItem is typically used to handle the first item in a page prior to processing the remainder in a loop
func (c *LogTailCollator) collectPageItem(it *azblob.FilterBlobItem) error {
	lt, err := NewLogTail(*it.Name)
	if err != nil {
		return err
	}
	// if it is missing, it will be the empty string that is set
	lastid := massifs.GetLastIDHex(collectTags(it.Tags))

	if lt.Ext == massifs.V1MMRMassifExt {
		cur, ok := c.Massifs[lt.Tenant]
		if !ok {
			lt.LastID = lastid
			c.Massifs[lt.Tenant] = lt
			return nil
		}
		if cur.TryReplaceTail(lt) {
			cur.LastID = lastid
			c.Massifs[lt.Tenant] = cur
		}
		return nil
	}

	// We only support 2 extensions, if we reach here we have excluded ".log" so
	// we know we have a seal
	cur, ok := c.Seals[lt.Tenant]
	if !ok {
		lt.LastID = lastid
		c.Seals[lt.Tenant] = lt
		return nil
	}
	if cur.TryReplaceTail(lt) {
		cur.LastID = lastid
		c.Seals[lt.Tenant] = cur
	}
	return nil
}

// CollatePage process a single page of azure blob filter results and collates
// the most recent LogTail's for each tenant represented in the page.
func (c *LogTailCollator) CollatePage(page []*azblob.FilterBlobItem) error {
	if len(page) == 0 {
		return nil
	}

	for _, it := range page {
		err := c.collectPageItem(it)
		if err != nil {
			return err
		}
	}
	return nil
}
