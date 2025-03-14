package massifs

import (
	"context"
	"crypto/ecdsa"

	"github.com/veraison/go-cose"
)

// IdentifiableCoseSigner represents a Cose1 signer that has additional methods to provide
// sufficient information to verify the signed product (an identifier for the signing key and the
// public key.)
type IdentifiableCoseSigner interface {
	cose.Signer
	PublicKey(ctx context.Context, kid string) (*ecdsa.PublicKey, error)
	LatestPublicKey() (*ecdsa.PublicKey, error)
	KeyIdentifier() string
	KeyLocation() string
}
