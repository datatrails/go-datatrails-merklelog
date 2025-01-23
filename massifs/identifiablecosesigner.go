package massifs

import (
	"crypto/ecdsa"

	"github.com/veraison/go-cose"
)

// IdentifiableCoseSigner represents a Cose1 signer that has additional methods to provide
// sufficient information to verify the signed product (an identifier for the signing key and the
// public key.)
type IdentifiableCoseSigner interface {
	cose.Signer
	PublicKey() (*ecdsa.PublicKey, error)
	KeyIdentifier() string
	KeyLocation() string
}
