package mmrblobs

import (
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"

	dtcbor "github.com/datatrails/go-datatrails-common/cbor"
	dtcose "github.com/datatrails/go-datatrails-common/cose"
	"github.com/ldclabs/cose/go/cwt"
	"github.com/veraison/go-cose"
)

var (
	ErrCurveNotSupported = errors.New("curve not supported")
)

type MMRState struct {
	// The size of the mmr defines the path to the root (and the full structure
	// of the tree). Note that all subsequent mmr states whose size is *greater*
	// than this, can also (efficiently) reproduce this particular root, and
	// hence can be used to verify 'old' receipts. This property is due to the
	// strict append only structure of the tree.
	MMRSize uint64 `cbor:"1,keyasint"`
	Root    []byte `cbor:"2,keyasint"`
	// Timestamp is the unix time read at the time the root was signed.
	// Including it allows for the same root to be re-signed.
	Timestamp int64 `cbor:"3,keyasint"`
}

type RootSigner struct {
	cborCodec   dtcbor.CBORCodec
	coseHeaders cose.Headers
	coseSigner  cose.Signer
}

type RootSignerConfig struct {
	Issuer        string
	Subject       string
	KeyIdentifier string
}

func NewRootSignerForECPrivateKey(
	cfg RootSignerConfig, key ecdsa.PrivateKey) (RootSigner, error) {

	alg, err := CoseAlgForEC(key.PublicKey)
	if err != nil {
		return RootSigner{}, nil
	}

	cnfClaim := NewCNFClaim(cfg.Issuer, cfg.Subject, cfg.KeyIdentifier, alg, key.PublicKey)

	codec, err := NewRootSignerCodec()
	if err != nil {
		return RootSigner{}, nil
	}

	signer, err := cose.NewSigner(alg, &key)
	if err != nil {
		return RootSigner{}, nil
	}

	rs := RootSigner{
		cborCodec: codec,
		coseHeaders: cose.Headers{
			Protected: cose.ProtectedHeader{
				dtcose.HeaderLabelCWTClaims: cnfClaim,
			},
		},
		coseSigner: signer,
	}
	return rs, nil
}

func (rs RootSigner) Sign1(state MMRState, external []byte) ([]byte, error) {
	payload, err := rs.cborCodec.MarshalCBOR(state)
	if err != nil {
		return nil, err
	}

	msg := cose.Sign1Message{
		Headers: rs.coseHeaders,
		Payload: payload,
	}
	err = msg.Sign(rand.Reader, external, rs.coseSigner)
	if err != nil {
		return nil, err
	}

	// We purposefully detach the root so that verifiers are forced to obtain it
	// from the log.
	state.Root = nil
	payload, err = rs.cborCodec.MarshalCBOR(state)
	if err != nil {
		return nil, err
	}
	msg.Payload = payload

	return msg.MarshalCBOR()
}

// CoseAlgForEC returns the appropraite algorithm for the provided public
// key curve or an error if the curve is not supported
//
// Noting that: "In order to promote interoperability, it is suggested that
// SHA-256 be used only with curve P-256, SHA-384 be used only with curve P-384,
// and SHA-512 be used with curve P-521." -- rfc 8152 & sec 4, 5480
func CoseAlgForEC(pub ecdsa.PublicKey) (cose.Algorithm, error) {

	switch pub.Curve.Params().Name {
	case "P-256":
		return cose.AlgorithmES256, nil
	case "P-384":
		return cose.AlgorithmES384, nil
	case "P-521":
		return cose.AlgorithmES512, nil
	default:
		return 0, fmt.Errorf("%s: %w", pub.Curve.Params().Name, ErrCurveNotSupported)
	}
}

func NewCNFClaim(
	issuer string, subject string, kid string, alg cose.Algorithm,
	pub ecdsa.PublicKey) map[int64]interface{} {

	claim := map[int64]interface{}{
		dtcose.CoseKeyLabel: map[int64]interface{}{
			dtcose.KeyIDLabel: kid,
			// XXX: TODO: we perversly use the wrong name in go-datatrails-common in order to use jwk / json. We need to change that, at least so that EC2 is accepted and returned in the cose context
			dtcose.KeyTypeLabel:   "EC", // EC2 is correct for rfc8152
			dtcose.AlgorithmLabel: alg,
			dtcose.ECCurveLabel:   pub.Curve.Params().Name,
			dtcose.ECXLabel:       pub.X.Bytes(),
			dtcose.ECYLabel:       pub.Y.Bytes(),
		},
	}
	return map[int64]interface{}{
		int64(cwt.KeyIss): issuer,
		int64(cwt.KeySub): subject,
		dtcose.CNFLabel:   claim,
	}
}

func NewRootSignerCodec() (dtcbor.CBORCodec, error) {
	codec, err := dtcbor.NewCBORCodec(
		dtcbor.NewDeterministicEncOpts(),
		dtcbor.NewDeterministicDecOpts(), // unsigned int decodes to uint64
	)
	if err != nil {
		return dtcbor.CBORCodec{}, err
	}
	return codec, nil
}

func newDecOptions() []dtcose.SignOption {
	return []dtcose.SignOption{dtcose.WithDecOptions(dtcbor.NewDeterministicDecOpts())}
}
