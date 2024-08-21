package massifs

import (
	"fmt"
	"strings"
)

const (
	V1MMRPrefix       = "v1/mmrs"
	V1MMRTenantPrefix = "v1/mmrs/tenant"

	V1MMRPathSep                     = "/"
	V1MMRExtSep                      = "."
	V1MMRMassifExt                   = "log"
	V1MMRBlobNameFmt                 = "%016d.log"
	V1MMRSignedTreeHeadBlobNameFmt   = "%016d.sth"
	V1MMRSealSignedRootExt           = "sth" // Signed Tree Head
	V1MMRConsistencyProofBlobNameFmt = "%016d.cproof"
	V1MMRSealCPROOF                  = "cproof" // Consistency Proof
	// LogInstanceN refers to the approach for handling blob size and format changes discussed at
	// [Changing the massifheight for a log](https://github.com/datatrails/epic-8120-scalable-proof-mechanisms/blob/1cb966cc10af03ae041fea4bca44b10979fb1eda/mmr/forestrie-mmrblobs.md#changing-the-massifheight-for-a-log)

	LogInstanceN = 0
)

// DataTrails Specifics of managing MMR's in azure blob storage

// TenantMassifPrefix return the path to the location of the massif blobs for
// the provided tenant identity. It is the callers responsibility to ensure the
// tenant identity has the correct form. 'tenant/uuid'
func TenantMassifPrefix(tenantIdentity string) string {
	return fmt.Sprintf(
		"%s/%s/%d/massifs/", V1MMRPrefix, tenantIdentity,
		LogInstanceN,
	)

}

// MassifPrefixForTenantUUID return the path to the location of the massif blobs
// for the provided tenant **UUID**. It is the callers responsibility to ensure
// the tenant uuid has the correct form. Ie, it is just the uuid in the
// appropriate form.
func MassifPrefixForTenantUUID(tenantUUID string) string {
	return fmt.Sprintf(
		"%s/%s/%d/massifs/", V1MMRTenantPrefix, tenantUUID,
		LogInstanceN,
	)
}

// TenantMassifSignedRootSPath returns the blob path for the log operator seals.
// The signatures and proofs necessary to associate the operator with the log
// and attest to its good operation.
func TenantMassifSignedRootsPrefix(tenantIdentity string) string {
	return fmt.Sprintf(
		"%s/%s/%d/massifseals/", V1MMRPrefix, tenantIdentity,
		LogInstanceN,
	)
}

// TenantMassifBlobPath returns the appropriate blob path for the blob
//
// The returned string forms a relative resource name with a versioned resource
// prefix of 'v1/mmrs/{tenant-identity}/massifs'
//
// Remembering that a legal {tenant-identity} has the form 'tenant/UUID'
//
// Because azure blob names and tags sort and compare only *lexically*, The
// number is represented in that path as a 16 digit hex string.
func TenantMassifBlobPath(tenantIdentity string, number uint64) string {
	return fmt.Sprintf(
		"%s%s", TenantMassifPrefix(tenantIdentity), fmt.Sprintf(V1MMRBlobNameFmt, number),
	)
}

// TenantRelativeMassifPath returns the blob path with the leading tenant identity and version schema striped.
func TenantRelativeMassifPath(tenantIdentity string, number uint32) string {
	return strings.TrimPrefix(
		strings.TrimPrefix(
			V1MMRPrefix, TenantMassifBlobPath(tenantIdentity, uint64(number))), tenantIdentity)
}

// TenantRelativeSealPath returns the seal path with the leading tenant identity and version schema striped.
func TenantRelativeSealPath(tenantIdentity string, number uint32) string {
	return strings.TrimPrefix(
		strings.TrimPrefix(
			V1MMRPrefix, TenantMassifSignedRootPath(tenantIdentity, number)), tenantIdentity)
}

// TenantMassifSignedRootPath returns the appropriate blob path for the blob
// root seal
//
// The returned string forms a relative resource name with a versioned resource
// prefix of 'v1/mmrs/{tenant-identity}/massifseals/'
//
// Remembering that a legal {tenant-identity} has the form 'tenant/UUID'
//
// Because azure blob names and tags sort and compare only *lexically*, The
// number is represented in that path as a 16 digit hex string.
func TenantMassifSignedRootPath(tenantIdentity string, massifIndex uint32) string {
	return fmt.Sprintf(
		"%s%s",
		TenantMassifSignedRootsPrefix(tenantIdentity),
		fmt.Sprintf(V1MMRSignedTreeHeadBlobNameFmt, massifIndex),
	)
}
