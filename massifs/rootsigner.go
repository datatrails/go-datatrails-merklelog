package massifs

import (
	"crypto/ecdsa"
	"crypto/rand"

	dtcbor "github.com/datatrails/go-datatrails-common/cbor"
	dtcose "github.com/datatrails/go-datatrails-common/cose"
	"github.com/veraison/go-cose"
)

// MMRState defines the details we include in our signed commitment to the head log state.
type MMRState struct {
	// The size of the mmr defines the path to the root (and the full structure
	// of the tree). Note that all subsequent mmr states whose size is *greater*
	// than this, can also (efficiently) reproduce this particular root, and
	// hence can be used to verify 'old' receipts. This property is due to the
	// strict append only structure of the tree.
	MMRSize uint64 `cbor:"1,keyasint"`
	Root    []byte `cbor:"2,keyasint"`
	// Timestamp is the unix time (milliseconds) read at the time the root was
	// signed. Including it allows for the same root to be re-signed.
	Timestamp int64 `cbor:"3,keyasint"`

	// Log configuration changes are (will be) applied from a particular log
	// leaf position a little like 'block height' co-ordination for ledgers. As
	// that configuration can impact how to interpret MMRSize (log data epochs
	// for example), we must attest to it in order to bind the state to a
	// specific log configuration. The applicable config is the first config
	// greater than EPOCH*IDTIMSTAMP. Any changes to things like massif height
	// or log format will result in configuration changes. This also allows for
	// logs to be chained by including the root from a previous log in a brand
	// new log. As epoch+idtimestamp is unique and continuous for the system.

	// Head leaf id timestamp and epoch. This is committed to by Root and is
	// taken from the last leaf in the log. The addition of which produced MMRSize.

	// The system unique timetamp value for the leaf that produced log MMRSize
	IDTimestamp uint64 `cbor:"4,keyasint"`

	// The current idtimestamp epoch (~17 year cadence. We use the unix epoch as
	// our base but roll twice as fast. so we are on epoch 1 in 2024)
	CommitmentEpoch uint32 `cbor:"6,keyasint"`
}

// RootSigner is used to produce a signature over an mmr log state.  This
// signature commits to a log state, and should only be created and published
// after checking the consistency between the last signed state and the new one.
// See merklelog/mmrblobs/logconfirmer.go:LogConfirmer for expected use.
type RootSigner struct {
	issuer    string
	cborCodec dtcbor.CBORCodec
}

func NewRootSigner(issuer string, cborCodec dtcbor.CBORCodec) RootSigner {
	rs := RootSigner{
		issuer:    issuer,
		cborCodec: cborCodec,
	}
	return rs
}

// Sign1 singes the provides state WARNING: You MUST check the state is
// consistent with the most recently signed state before publishing this with a
// datatrails signature.
func (rs RootSigner) Sign1(coseSigner cose.Signer, keyIdentifier string, publicKey *ecdsa.PublicKey, subject string, state MMRState, external []byte) ([]byte, error) {
	payload, err := rs.cborCodec.MarshalCBOR(state)
	if err != nil {
		return nil, err
	}

	coseHeaders := cose.Headers{
		Protected: cose.ProtectedHeader{
			dtcose.HeaderLabelCWTClaims: dtcose.NewCNFClaim(
				rs.issuer, subject, keyIdentifier, coseSigner.Algorithm(), *publicKey),
		},
	}

	msg := cose.Sign1Message{
		Headers: coseHeaders,
		Payload: payload,
	}
	err = msg.Sign(rand.Reader, external, coseSigner)
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
