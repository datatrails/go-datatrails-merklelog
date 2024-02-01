package mmrblobs

import "fmt"

const (
	V1MMRPrefix      = "v1/mmrs"
	V1MMRBlobNameFmt = "%016d.log"
)

// DataTrails Specifics of managing MMR's in azure blob storage

func TenantMassifPrefix(tenantIdentity string) string {
	return fmt.Sprintf(
		"%s/%s/massifs/", V1MMRPrefix, tenantIdentity,
	)

}

// TenantEpochMountainBlobPath returns the appropriate blob path for the blob
//
// We partition the blob space conveniently for working with the double batched
// merkle log accumulator scheme described  by
// Justin Drake [here](https://ethresear.ch/t/double-batched-merkle-log-accumulator/571)
//
// The returned string forms a relative resource name with a versioned resource
// prefix of 'v1/mmrs'
//
// Because azure blob names and tags sort and compare only *lexically*, The
// number is represented in that path as a 16 digit hex string.
func TenantMassifBlobPath(tenantIdentity string, number uint64) string {
	return fmt.Sprintf(
		"%s%s", TenantMassifPrefix(tenantIdentity), fmt.Sprintf(V1MMRBlobNameFmt, number),
	)
}
