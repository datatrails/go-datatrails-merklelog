package mmrblobs

import "fmt"

const (
	V1MMRPrefix                      = "v1/mmrs"
	V1MMRBlobNameFmt                 = "%016d.log"
	V1MMRSignedTreeHeadBlobNameFmt   = "%016d.sth"
	V1MMSealSignedRoot               = "sth" // Signed Tree Head
	V1MMRConsistencyProofBlobNameFmt = "%016d.cproof"
	V1MMRSealCPROOF                  = "cproof" // Consistency Proof
)

// DataTrails Specifics of managing MMR's in azure blob storage

func TenantMassifPrefix(tenantIdentity string) string {
	return fmt.Sprintf(
		"%s/%s/massifs/", V1MMRPrefix, tenantIdentity,
	)

}

// TenantMassifSignedRootSPath returns the blob path for the log operator seals.
// The signatures and proofs necessary to associate the operator with the log
// and attest to its good operation.
func TenantMassifSignedRootsPrefix(tenantIdentity string) string {
	return fmt.Sprintf(
		"%s/%s/signedroots/", V1MMRPrefix, tenantIdentity,
	)
}

// TenantMassifBlobPath returns the appropriate blob path for the blob
//
// The returned string forms a relative resource name with a versioned resource
// prefix of 'v1/mmrs/{tenant-identity}/massifs'
//
// Because azure blob names and tags sort and compare only *lexically*, The
// number is represented in that path as a 16 digit hex string.
func TenantMassifBlobPath(tenantIdentity string, number uint64) string {
	return fmt.Sprintf(
		"%s%s", TenantMassifPrefix(tenantIdentity), fmt.Sprintf(V1MMRBlobNameFmt, number),
	)
}

// TenantMassifSignedRootPath returns the appropriate blob path for the blob
// root seal
//
// The returned string forms a relative resource name with a versioned resource
// prefix of 'v1/mmrs/{tenant-identity}/signedroots/'
//
// Because azure blob names and tags sort and compare only *lexically*, The
// number is represented in that path as a 16 digit hex string.
func TenantMassifSignedRootPath(tenantIdentity string, number uint64) string {
	return fmt.Sprintf(
		"%s%s.%s",
		TenantMassifSignedRootsPrefix(tenantIdentity),
		fmt.Sprintf(V1MMRSignedTreeHeadBlobNameFmt, number),
		V1MMSealSignedRoot,
	)
}
