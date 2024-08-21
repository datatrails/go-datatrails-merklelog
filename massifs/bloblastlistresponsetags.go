package massifs

// this file exists to avoid an ugly package alias

import (
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

func listResponseTags(blobTags *azblob.BlobTags) map[string]string {
	if blobTags == nil {
		return nil
	}
	tags := map[string]string{}
	for _, tag := range blobTags.BlobTagSet {
		if tag.Key == nil || tag.Value == nil {
			continue
		}
		tags[*tag.Key] = *tag.Value
	}
	return tags
}
