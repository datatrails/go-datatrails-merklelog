package massifs

import (
	"crypto/elliptic"
	"testing"

	"github.com/datatrails/go-datatrails-common/azkeys"
	dtcose "github.com/datatrails/go-datatrails-common/cose"
	"github.com/datatrails/go-datatrails-common/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootSigner_Sign1(t *testing.T) {

	logger.New("TEST")

	type fields struct {
		issuer string
		kid    string
		curve  elliptic.Curve
	}
	type args struct {
		subject  string
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
				issuer: "synsation.org",
				kid:    "log attestation key 1",
				curve:  elliptic.P256(),
			},
			args: args{
				subject: "merklelog-attestor",
				state: MMRState{
					MMRSize:   1,
					Peaks:     [][]byte{{1}},
					Timestamp: 1234,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			key := TestGenerateECKey(t, elliptic.P256())
			rs := TestNewRootSigner(t, tt.fields.issuer)

			coseSigner := azkeys.NewTestCoseSigner(t, key)
			pubKey, err := coseSigner.PublicKey()
			require.NoError(t, err)

			coseMsg, err := rs.Sign1(coseSigner, coseSigner.KeyIdentifier(), pubKey, tt.args.subject, tt.args.state, tt.args.external)
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
			state.Peaks = tt.args.state.Peaks
			err = VerifySignedRoot(
				rs.cborCodec,
				dtcose.NewCWTPublicKeyProvider(signed),
				signed, state, nil,
			)

			assert.NoError(t, err)
		})
	}
}
