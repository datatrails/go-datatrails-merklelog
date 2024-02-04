package mmrblobs

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"reflect"
	"testing"

	dtcose "github.com/datatrails/go-datatrails-common/cose"
	"github.com/datatrails/go-datatrails-common/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/go-cose"
)

func TestCoseAlgForEC(t *testing.T) {
	type args struct {
		pub ecdsa.PublicKey
	}
	tests := []struct {
		name    string
		args    args
		want    cose.Algorithm
		wantErr bool
	}{
		{
			name: "P-256 ok",
			args: args{
				ecdsa.PublicKey{
					Curve: &elliptic.CurveParams{
						Name: "P-256",
					},
				},
			},
			want: cose.AlgorithmES256,
		},
		{
			name: "P-384 ok",
			args: args{
				ecdsa.PublicKey{
					Curve: &elliptic.CurveParams{
						Name: "P-384",
					},
				},
			},
			want: cose.AlgorithmES384,
		},
		{
			name: "P-521 ok",
			args: args{
				ecdsa.PublicKey{
					Curve: &elliptic.CurveParams{
						Name: "P-521",
					},
				},
			},
			want: cose.AlgorithmES512,
		},
		{
			name: "P-512 NOT ok",
			args: args{
				ecdsa.PublicKey{
					Curve: &elliptic.CurveParams{
						Name: "P-512",
					},
				},
			},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CoseAlgForEC(tt.args.pub)
			if (err != nil) != tt.wantErr {
				t.Errorf("CoseAlgForEC() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CoseAlgForEC() = %v, want %v", got, tt.want)
			}
		})
	}
}

func mustNewRootSigner(t *testing.T, cfg RootSignerConfig, key ecdsa.PrivateKey) RootSigner {
	rs, err := NewRootSignerForECPrivateKey(cfg, key)
	require.NoError(t, err)
	return rs
}

func mustGenerateECKey(t *testing.T, curve elliptic.Curve) ecdsa.PrivateKey {
	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	require.NoError(t, err)
	return *privateKey
}

func TestRootSigner_Sign1(t *testing.T) {

	logger.New("TEST")

	type fields struct {
		issuer  string
		subject string
		kid     string
		curve   elliptic.Curve
	}
	type args struct {
		state    MMRState
		external []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "common case P-256 & ES256",
			fields: fields{
				issuer:  "synsation.org",
				subject: "merklelog-attestor",
				kid:     "log attestation key 1",
				curve:   elliptic.P256(),
			},
			args: args{
				state: MMRState{
					MMRSize:   1,
					Root:      []byte{1},
					Timestamp: 1234,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			key := mustGenerateECKey(t, elliptic.P256())
			rs := mustNewRootSigner(t,
				RootSignerConfig{
					Issuer:        tt.fields.issuer,
					Subject:       tt.fields.subject,
					KeyIdentifier: tt.fields.kid,
				}, key)
			coseMsg, err := rs.Sign1(tt.args.state, tt.args.external)
			if (err != nil) != tt.wantErr {
				t.Errorf("RootSigner.Sign1() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			signed, state, err := DecodeSignedRoot(rs.cborCodec, coseMsg)
			assert.NoError(t, err)

			err = VerifySignedRoot(
				rs.cborCodec,
				dtcose.NewCWTPublicKeyProvider(signed),
				signed, state, nil,
			)
			// verification must fail if we haven't put the root in
			assert.Error(t, err)

			// This is step 2. Usually we would work out the massif, read that
			// blob then compute the root from it by passing MMRState.MMRSize to
			// GetRoot
			state.Root = tt.args.state.Root
			err = VerifySignedRoot(
				rs.cborCodec,
				dtcose.NewCWTPublicKeyProvider(signed),
				signed, state, nil,
			)

			assert.NoError(t, err)
		})
	}
}
