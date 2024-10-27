package massifs

import (
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/datatrails/go-datatrails-common/azkeys"
	commoncose "github.com/datatrails/go-datatrails-common/cose"
	_ "github.com/fxamacker/cbor/v2"
	"github.com/veraison/go-cose"
	_ "github.com/veraison/go-cose"

	"github.com/datatrails/go-datatrails-common/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCoseSign1_UnprotectedEncDec just checks our asumptions about how to encode and decode
// nested cose messages in the unprotected headers of a cose sign1 message.
// There are some gotcha's in the encoding rules when nesting cose messages and this test is used
// to isolate the aspects we care about for the MMRIVER pre-signed receipts.
func TestCoseSign1_UnprotectedEncDec(t *testing.T) {
	logger.New("TEST")

	key := TestGenerateECKey(t, elliptic.P256())
	cborCodec, err := NewRootSignerCodec()
	require.NoError(t, err)
	coseSigner := azkeys.NewTestCoseSigner(t, key)
	rs := TestNewRootSigner(t, "test-issuer")

	mustMarshalCBOR := func(value any) []byte {
		b, err := cborCodec.MarshalCBOR(value)
		require.NoError(t, err)
		return b
	}

	mustSignPeak := func(peak []byte) []byte {
		b, err := rs.signEmptyPeakReceipt(coseSigner, &key.PublicKey, "test-key", "test-issuer", "test-subject", peak)
		require.NoError(t, err)
		return b
	}

	mustSignPeaks := func(peaks [][]byte) [][]byte {
		receipts, err := rs.signEmptyPeakReceipts(coseSigner, &key.PublicKey, "test-key", "test-issuer", "test-subject", peaks)
		require.NoError(t, err)
		return receipts
	}

	mustSignMessage := func(payload []byte, headers cose.Headers) []byte {

		headers.Protected[commoncose.HeaderLabelCWTClaims] = commoncose.NewCNFClaim(
			"test-issuer", "test-subject", "test-key", coseSigner.Algorithm(),
			key.PublicKey,
		)

		msg := cose.Sign1Message{
			Headers: headers,
			Payload: payload,
		}
		err := msg.Sign(rand.Reader, nil, coseSigner)
		require.NoError(t, err)

		encodable, err := commoncose.NewCoseSign1Message(&msg)
		require.NoError(t, err)
		encoded, err := encodable.MarshalCBOR()
		require.NoError(t, err)
		return encoded
	}

	verifyDecoded := func(decoded *commoncose.CoseSign1Message) error {
		_, ok := decoded.Headers.Protected[commoncose.HeaderLabelCWTClaims]
		if ok {
			return decoded.VerifyWithCWTPublicKey(nil)
		}
		return decoded.VerifyWithPublicKey(&key.PublicKey, nil)
	}

	testDecodVerify := func(encoded []byte, t *testing.T) {
		decoded, err := commoncose.NewCoseSign1MessageFromCBOR(encoded)
		assert.NoError(t, err)

		err = verifyDecoded(decoded)
		assert.NoError(t, err)
	}

	testDecodeSingleNestedVerify := func(encoded []byte, t *testing.T) {

		var err error
		var decoded *commoncose.CoseSign1Message
		decoded, err = commoncose.NewCoseSign1MessageFromCBOR(encoded)
		assert.NoError(t, err)

		err = verifyDecoded(decoded)
		assert.NoError(t, err)

		singleNested, ok := decoded.Headers.Unprotected[int64(-65535-1)]
		assert.True(t, ok)
		if !ok {
			return
		}
		b, ok := singleNested.([]byte)
		assert.True(t, ok)
		if !ok {
			return
		}
		decoded, err = commoncose.NewCoseSign1MessageFromCBOR(b)
		assert.NoError(t, err)
		err = verifyDecoded(decoded)
		assert.NoError(t, err)
		return
	}

	testDecodeArrayOfNestedVerify := func(encoded []byte, t *testing.T) {

		var err error
		var decoded *commoncose.CoseSign1Message
		decoded, err = commoncose.NewCoseSign1MessageFromCBOR(encoded)
		assert.NoError(t, err)
		err = verifyDecoded(decoded)
		assert.NoError(t, err)

		arrayOfNested, ok := decoded.Headers.Unprotected[int64(-65535-2)]
		assert.True(t, ok)
		if !ok {
			return
		}
		outer, ok := arrayOfNested.([]interface{})
		assert.True(t, ok)
		for _, inner := range outer {
			b, ok := inner.([]byte)
			assert.True(t, ok)
			if !ok {
				return
			}
			decoded, err := commoncose.NewCoseSign1MessageFromCBOR(b)
			assert.NoError(t, err)
			err = verifyDecoded(decoded)
			assert.NoError(t, err)
		}
	}

	// TestDecode is a test case specific decoder test function
	type TestDecode func(encoded []byte, t *testing.T)

	type fields struct {
		Protected   cose.ProtectedHeader
		Unprotected cose.UnprotectedHeader
		Payload     []byte
	}
	tests := []struct {
		name       string
		fields     fields
		testDecode TestDecode
	}{
		{
			name: "cbor payload, unprotected header with private range array of signed peaks",
			fields: fields{
				Protected: cose.ProtectedHeader{
					"alg": coseSigner.Algorithm(),
					"kid": "log attestation key 1",
				},
				Unprotected: cose.UnprotectedHeader{
					-65535 - 0: mustSignPeaks([][]byte{{
						0, 1, 2, 3, 4, 5, 6, 7,
						8, 9, 10, 11, 12, 13, 14, 15,
						16, 17, 18, 19, 20, 21, 22, 23,
						24, 25, 26, 27, 28, 29, 30, 31,
					}, {
						0, 1, 2, 3, 4, 5, 6, 7,
						8, 9, 10, 11, 12, 13, 14, 15,
						16, 17, 18, 19, 20, 21, 22, 23,
						24, 25, 26, 27, 28, 29, 30, 31,
					},
					}),
				},
				Payload: mustMarshalCBOR(MMRState{
					MMRSize: 1,
					Peaks: [][]byte{{
						0, 1, 2, 3, 4, 5, 6, 7,
						8, 9, 10, 11, 12, 13, 14, 15,
						16, 17, 18, 19, 20, 21, 22, 23,
						24, 25, 26, 27, 28, 29, 30, 31,
					}},
					Timestamp: 1234,
				}),
			},
			testDecode: testDecodVerify,
		},

		{
			name: "cbor payload, unprotected header with private range signed peak",
			fields: fields{
				Protected: cose.ProtectedHeader{
					"alg": coseSigner.Algorithm(),
					"kid": "log attestation key 1",
				},
				Unprotected: cose.UnprotectedHeader{
					-65535 - 0: mustSignPeak([]byte{
						0, 1, 2, 3, 4, 5, 6, 7,
						8, 9, 10, 11, 12, 13, 14, 15,
						16, 17, 18, 19, 20, 21, 22, 23,
						24, 25, 26, 27, 28, 29, 30, 31,
					}),
				},
				Payload: mustMarshalCBOR(MMRState{
					MMRSize: 1,
					Peaks: [][]byte{{
						0, 1, 2, 3, 4, 5, 6, 7,
						8, 9, 10, 11, 12, 13, 14, 15,
						16, 17, 18, 19, 20, 21, 22, 23,
						24, 25, 26, 27, 28, 29, 30, 31,
					}},
					Timestamp: 1234,
				}),
			},
			testDecode: testDecodVerify,
		},

		{
			name: "cbor payload, unprotected header with private range integer value",
			fields: fields{
				Protected: cose.ProtectedHeader{
					"alg": coseSigner.Algorithm(),
					"kid": "log attestation key 1",
				},
				Unprotected: cose.UnprotectedHeader{
					-65535 - 0: 123,
				},
				Payload: mustMarshalCBOR(MMRState{
					MMRSize: 1,
					Peaks: [][]byte{{
						0, 1, 2, 3, 4, 5, 6, 7,
						8, 9, 10, 11, 12, 13, 14, 15,
						16, 17, 18, 19, 20, 21, 22, 23,
						24, 25, 26, 27, 28, 29, 30, 31}},
					Timestamp: 1234,
				}),
			},
			testDecode: testDecodVerify,
		},

		{
			name: "unprotected header with private range nested signed message",
			fields: fields{
				Protected: cose.ProtectedHeader{
					"alg": coseSigner.Algorithm(),
					"kid": "log attestation key 1",
				},
				Unprotected: cose.UnprotectedHeader{
					-65535 - 1: mustSignMessage([]byte("hello continent"), cose.Headers{
						Protected: cose.ProtectedHeader{
							"alg": coseSigner.Algorithm(),
							"kid": "log attestation key 1",
						},
					}),
				},
				Payload: []byte("hello world"),
			},
			testDecode: testDecodeSingleNestedVerify,
		},
		{
			name: "unprotected header with private range nested signed message",
			fields: fields{
				Protected: cose.ProtectedHeader{
					"alg": coseSigner.Algorithm(),
					"kid": "log attestation key 1",
				},
				Unprotected: cose.UnprotectedHeader{
					-65535 - 2: [][]byte{
						mustSignMessage([]byte("hello uk"), cose.Headers{
							Protected: cose.ProtectedHeader{
								"alg": coseSigner.Algorithm(),
								"kid": "log attestation key 1",
							},
						}),
						mustSignMessage([]byte("hello france"), cose.Headers{
							Protected: cose.ProtectedHeader{
								"alg": coseSigner.Algorithm(),
								"kid": "log attestation key 1",
							},
						}),
					},
				},
				Payload: []byte("hello world"),
			},
			testDecode: testDecodeArrayOfNestedVerify,
		},

		{
			name: "empty unprotected headers",
			fields: fields{
				Protected: cose.ProtectedHeader{
					"alg": coseSigner.Algorithm(),
					"kid": "log attestation key 1",
				},
				Unprotected: cose.UnprotectedHeader{},
				Payload:     []byte("hello world"),
			},
			testDecode: testDecodVerify,
		},
		{
			name: "unprotected header with private range integer value",
			fields: fields{
				Protected: cose.ProtectedHeader{
					"alg": coseSigner.Algorithm(),
					"kid": "log attestation key 1",
				},
				Unprotected: cose.UnprotectedHeader{
					-65535 - 0: 123,
				},
				Payload: []byte("hello world"),
			},
			testDecode: testDecodVerify,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var err error

			// cborCodec, err := NewRootSignerCodec()
			// require.NoError(t, err)

			headers := cose.Headers{
				Protected:   tt.fields.Protected,
				Unprotected: tt.fields.Unprotected,
			}

			msg := cose.Sign1Message{
				Headers: headers,
				Payload: tt.fields.Payload,
			}
			err = msg.Sign(rand.Reader, nil, coseSigner)
			require.NoError(t, err)

			encodable, err := commoncose.NewCoseSign1Message(&msg)
			assert.NoError(t, err)
			encoded, err := encodable.MarshalCBOR()
			assert.NoError(t, err)

			if tt.testDecode != nil {
				tt.testDecode(encoded, t)
			}
		})
	}
}

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
					MMRSize: 1,
					Peaks: [][]byte{{
						0, 1, 2, 3, 4, 5, 6, 7,
						8, 9, 10, 11, 12, 13, 14, 15,
						16, 17, 18, 19, 20, 21, 22, 23,
						24, 25, 26, 27, 28, 29, 30, 31}},
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

			err = VerifySignedCheckPoint(
				rs.cborCodec,
				commoncose.NewCWTPublicKeyProvider(signed),
				signed, state, nil,
			)
			// verification must fail if we haven't put the root in
			assert.Error(t, err)

			// This is step 2. Usually we would work out the massif, read that
			// blob then compute the root from it by passing MMRState.MMRSize to
			// GetRoot
			state.Peaks = tt.args.state.Peaks
			err = VerifySignedCheckPoint(
				rs.cborCodec,
				commoncose.NewCWTPublicKeyProvider(signed),
				signed, state, nil,
			)

			assert.NoError(t, err)
		})
	}
}
