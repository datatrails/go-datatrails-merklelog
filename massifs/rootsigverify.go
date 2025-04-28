package massifs

import (
	"crypto"

	"github.com/datatrails/go-datatrails-common/cbor"
	dtcose "github.com/datatrails/go-datatrails-common/cose"
	"github.com/veraison/go-cose"
)

type publicKeyProvider interface {
	PublicKey() (crypto.PublicKey, cose.Algorithm, error)
}

// DecodeSignedRoot decodes the MMRState values from the signed message
// See VerifySignedCheckPoint for a description of how to verify a signed root
func DecodeSignedRoot(
	codec cbor.CBORCodec, msg []byte,
) (*dtcose.CoseSign1Message, MMRState, error) {
	signed, err := dtcose.NewCoseSign1MessageFromCBOR(msg, newCheckpointDecOptions()...)
	if err != nil {
		return nil, MMRState{}, err
	}

	var unverifiedState MMRState
	err = codec.UnmarshalInto(signed.Payload, &unverifiedState)
	if err != nil {
		return nil, MMRState{}, err
	}
	return signed, unverifiedState, nil
}

// VerifySignedCheckPoint applies the provided state to the signed message and
// verifies the result
//
// When signing and publishing roots, we remove the peaks from the signed message
// prior to publishing. So that it can only be verified by recovering the peaks
// from the mmr at the size in the signed message.
//
// Verification of a signed root is a 3 step process:
//  1. Use DecodeSignedRoot to obtain the MMRState from the signed message. This
//     state will not verify as the peaks have been removed after signing.
//  2. Use MMRState.MMRSize to obtain the peaks of the log corresponding to that size
//  3. Update the MMRState with the derived peaks and call this function to complete the verification
func VerifySignedCheckPoint(
	codec cbor.CBORCodec, keyProvider publicKeyProvider, signed *dtcose.CoseSign1Message, unverifiedState MMRState, external []byte) error {

	var err error
	signed.Payload, err = codec.MarshalCBOR(unverifiedState)
	if err != nil {
		return err
	}
	return signed.VerifyWithProvider(keyProvider, external)
}
