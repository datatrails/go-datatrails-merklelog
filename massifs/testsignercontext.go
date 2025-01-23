package massifs

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"
	"testing"

	"github.com/datatrails/go-datatrails-common/azkeys"
	"github.com/datatrails/go-datatrails-common/cbor"
	"github.com/datatrails/go-datatrails-common/cose"
	"github.com/stretchr/testify/assert"
)

type TestSignerContext struct {
	Key             ecdsa.PrivateKey
	RootSigner      RootSigner
	CoseSigner      *azkeys.TestCoseSigner
	RootSignerCodec cbor.CBORCodec
}

func NewTestSignerContext(t *testing.T, issuer string) *TestSignerContext {
	var err error

	key := TestGenerateECKey(t, elliptic.P256())
	s := &TestSignerContext{
		Key:        key,
		RootSigner: TestNewRootSigner(t, issuer),
		CoseSigner: azkeys.NewTestCoseSigner(t, key),
	}
	s.RootSignerCodec, err = NewRootSignerCodec()
	assert.NoError(t, err)

	return s
}

func (s *TestSignerContext) SignedState(
	tenantIdentity string, massifIndex uint64, state MMRState,
) (*cose.CoseSign1Message, MMRState, error) {
	subject := TenantMassifBlobPath(tenantIdentity, massifIndex)
	data, err := signState(s.RootSigner, s.CoseSigner, subject, state)
	if err != nil {
		return nil, MMRState{}, err
	}
	return DecodeSignedRoot(s.RootSignerCodec, data)
}

func (s *TestSignerContext) SealedState(tenantIdentity string, massifIndex uint64, state MMRState) (*SealedState, error) {
	signed, state, err := s.SignedState(tenantIdentity, massifIndex, state)
	if err != nil {
		return nil, err
	}
	return &SealedState{
		Sign1Message: *signed,
		MMRState:     state,
	}, nil
}

func signState(
	rootSigner RootSigner,
	coseSigner IdentifiableCoseSigner,
	subject string,
	state MMRState,
) ([]byte, error) {

	publicKey, err := coseSigner.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("unable to get public key for signing key %w", err)
	}

	keyIdentifier := coseSigner.KeyIdentifier()
	data, err := rootSigner.Sign1(coseSigner, keyIdentifier, publicKey, subject, state, nil)
	if err != nil {
		return nil, err
	}
	return data, nil
}
