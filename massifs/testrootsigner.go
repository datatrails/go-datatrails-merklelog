package massifs

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateECKey(t *testing.T, curve elliptic.Curve) ecdsa.PrivateKey {
	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	require.NoError(t, err)
	return *privateKey
}

func TestNewRootSigner(t *testing.T, issuer string) RootSigner {
	cborCodec, err := NewRootSignerCodec()
	require.NoError(t, err)
	rs := NewRootSigner(issuer, cborCodec)
	return rs
}
