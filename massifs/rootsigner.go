package massifs

import (
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"

	commoncbor "github.com/datatrails/go-datatrails-common/cbor"
	commoncose "github.com/datatrails/go-datatrails-common/cose"
	"github.com/fxamacker/cbor/v2"
	"github.com/veraison/go-cose"
)

var (
	ErrNodeSize = errors.New("node value sizes must match the hash size")
)

type MMRStateVersion int

const (
	MMRStateVersion0 MMRStateVersion = iota // Implicit initial release version
	MMRStateVersion1                        // Version 1
	// Note: new versions must be monotonicaly assigned.

)

const (
	MMRStateVersionCurrent  = MMRStateVersion1
	VDSCoseReceiptsTag      = 395
	VDSCoseReceiptProofsTag = 396
	VDSMMRiver              = 2
	VDSInclusionProof       = -1
	InclusionProofIndex     = 1
	InclusionProofProof     = 2

	// The numbers < -65535 are reserved for private use.
	COSEPrivateStart = int64(-65535)
	// Numbers in the private use space are organisation / implementation specific.
	// Allocation in this range MUST be co-ordinated datatrails wide.
	// Remembering that the range is *negative* we allocate the tag by
	// subtracting the IANA registered tag for marking COSE Receipts proof data.
	SealPeakReceiptsLabel = int64(COSEPrivateStart - VDSCoseReceiptProofsTag)
)

// MMRState defines the details we include in our signed commitment to the head log state.
type MMRState struct {

	// Version is present in all seals from version 1. The initial release was implicity version 0.
	Version int `cbor:"7,keyasint"`

	// The size of the mmr defines the path to the root (and the full structure
	// of the tree). Note that all subsequent mmr states whose size is *greater*
	// than this, can also (efficiently) reproduce this particular root, and
	// hence can be used to verify 'old' receipts. This property is due to the
	// strict append only structure of the tree.
	MMRSize        uint64 `cbor:"1,keyasint"`
	LegacySealRoot []byte `cbor:"2,keyasint"` //  Valid in Version 0 only.
	// The peak hashes for the mmr identified by MMRSize, this is also the packed accumulator for the tree state.
	// All inclusion proofs for any node under MMRSize will lead directly to one
	// of these peaks, or can be extended to do so.
	Peaks [][]byte `cbor:"8,keyasint"` // Version 1+
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

type MMRStateReceipts struct {
	// A Pre-signed COSE Receipts MMRIVER COSE_Sign1 message for each peak in the MMR identified by MMRSize.
	// To create a receipt, simply attach the inclusion proof to the unprotected header for the appropriate PeakIndex.
	// PeakReceipts []cbor.RawMessage `cbor:"-65931,keyasint"`
	PeakReceipts [][]byte `cbor:"-65931,keyasint"`
}

// RootSigner is used to produce a signature over an mmr log state.  This
// signature commits to a log state, and should only be created and published
// after checking the consistency between the last signed state and the new one.
// See merklelog/mmrblobs/logconfirmer.go:LogConfirmer for expected use.
type RootSigner struct {
	issuer    string
	cborCodec commoncbor.CBORCodec
}

func NewRootSigner(issuer string, cborCodec commoncbor.CBORCodec) RootSigner {
	rs := RootSigner{
		issuer:    issuer,
		cborCodec: cborCodec,
	}
	return rs
}

// Sign1 singes the provides state WARNING: You MUST check the state is
// consistent with the most recently signed state before publishing this with a
// datatrails signature.
func (rs RootSigner) Sign1(
	coseSigner cose.Signer,
	keyIdentifier string,
	publicKey *ecdsa.PublicKey,
	subject string,
	state MMRState, external []byte) ([]byte, error) {

	receipts, err := rs.signEmptyPeakReceipts(coseSigner, publicKey, keyIdentifier, rs.issuer, subject, state.Peaks)
	if err != nil {
		return nil, err
	}
	if len(receipts) != len(state.Peaks) {
		return nil, fmt.Errorf("receipt vs peak count mismatch: %d vs %d", len(receipts), len(state.Peaks))
	}

	coseHeaders := cose.Headers{
		Protected: cose.ProtectedHeader{
			commoncose.HeaderLabelCWTClaims: commoncose.NewCNFClaim(
				rs.issuer, subject, keyIdentifier, coseSigner.Algorithm(), *publicKey),
		},
		// one receipt is present for each peak identified by tree-size-2 in
		// the protected header each receipt is individualy signed
		// COSE_Sign1 message over that specific peak. All receipts of
		// inclusion for individual leaves are created by attaching proofs
		// to the unprotected header of the peak receipt.
		// SealPeakReceiptsLabel: receipts,
		// RawUnprotected: rawunprotected,
		Unprotected: cose.UnprotectedHeader{
			SealPeakReceiptsLabel: receipts,
		},
	}

	payload, err := rs.cborCodec.MarshalCBOR(state)
	if err != nil {
		return nil, err
	}

	msg := cose.Sign1Message{
		Headers: coseHeaders,
		Payload: payload,
	}
	err = msg.Sign(rand.Reader, external, coseSigner)
	if err != nil {
		return nil, err
	}

	// We purposefully detach the peaks so that verifiers are forced to obtain it
	// from the log.
	state.LegacySealRoot = nil
	state.Peaks = nil

	payload, err = rs.cborCodec.MarshalCBOR(state)
	if err != nil {
		return nil, err
	}

	msg.Payload = payload

	encodable, err := commoncose.NewCoseSign1Message(&msg)
	if err != nil {
		return nil, err
	}
	return encodable.MarshalCBOR()
}

// Note: regarding why and how we can pre-sign receipts:
//
// A specific advantage of MMR's is that we can pre-sign the protected headers
// for all receipts we will ever be asked for. The scitt endpoint then only has
// to copy the pre-signed receipt and *add* the inclusion path it is asked for.
//
// Importantly, this allows for self service *privacy preserving*, scitt
// compatible, receipts based on replicated copies of the log.
//
// The most natural place to produce the pre-signed receipts is in the the log
// confirmer, because we are allways pre-signing *peaks* of the MMR. And the
// consistency between peaks (accumulators) is the concern of the sealer by way
// of LogConfirmer. And the most natural place to store them is in the massif seal.
// Whis is what we accomodate here.
func (c *RootSigner) signEmptyPeakReceipts(
	coseSigner cose.Signer,
	publicKey *ecdsa.PublicKey,
	keyIdentifier string,
	issuer string,
	subject string,
	peaks [][]byte,
) ([][]byte, error) {

	receipts := make([][]byte, len(peaks))

	for i, peak := range peaks {
		receipt, err := c.signEmptyPeakReceipt(coseSigner, publicKey, keyIdentifier, issuer, subject, peak)
		if err != nil {
			return nil, err
		}

		receipts[i] = receipt
	}
	return receipts, nil
}

// signEmptyPeakReceipt signes a Receipt for an accumulator peak.
//
// Because many inclusion proofs lead to the same peak, the proof material for
// the unprotected header is empty. This can be added by the log consumer in a
// privacy preserving way based on replicated massif content.
//
// Arguments:
//
//	  ctx: The context for the operation
//	  coseSigner: The signer of the completed shared receipt
//	  issuer: The identifier for the issuer of the receipt
//		 subject: The identifier for the subject of the receipt
func (rs RootSigner) signEmptyPeakReceipt(
	coseSigner cose.Signer,
	publicKey *ecdsa.PublicKey,
	keyIdentifier string,
	issuer string,
	subject string,
	// The bytes of a peak, which an mmr node which is a member of an accumulator for one or more tree states.
	peak []byte,
) ([]byte, error) {

	if len(peak) != 32 {
		return nil, fmt.Errorf("%w: peak must be 32 bytes, got %d", ErrNodeSize, len(peak))
	}

	headers := cose.Headers{
		Protected: cose.ProtectedHeader{
			VDSCoseReceiptsTag:        VDSMMRiver,
			cose.HeaderLabelAlgorithm: coseSigner.Algorithm(),
			cose.HeaderLabelKeyID:     []byte(keyIdentifier),
			commoncose.HeaderLabelCWTClaims: commoncose.NewCNFClaim(
				issuer,
				subject,
				keyIdentifier,
				coseSigner.Algorithm(),
				*publicKey),
		},
		// The receipt producer, which MAY be the relying party in possesion of
		// a log massif, can fill in the inclusion proof directly and
		// independently, without revealing the item of interest to the log
		// service.
		Unprotected: cose.UnprotectedHeader{},
	}

	msg := cose.Sign1Message{
		Headers: headers,
		Payload: peak,
	}

	err := msg.Sign(rand.Reader, nil, coseSigner)
	if err != nil {
		return nil, err
	}

	// now, detach the payload
	msg.Payload = nil

	// Use the appropraite encoding options
	encodable, err := commoncose.NewCoseSign1Message(&msg)
	if err != nil {
		return nil, err
	}
	return encodable.MarshalCBOR()
}

func NewRootSignerCodec() (commoncbor.CBORCodec, error) {
	codec, err := commoncbor.NewCBORCodec(encOptions, decOptions)
	if err != nil {
		return commoncbor.CBORCodec{}, err
	}
	return codec, nil
}

var (
	encOptions = commoncbor.NewDeterministicEncOpts()
	decOptions = cbor.DecOptions{
		DupMapKey:   cbor.DupMapKeyEnforcedAPF, // (default) duplicated key not allowed
		IndefLength: cbor.IndefLengthForbidden, // (default) no streaming
		// override the default decoding behaviour for unsigned integers to retain the sign
		IntDec: cbor.IntDecConvertNone, // decode CBOR uint/int to Go int64
		TagsMd: cbor.TagsForbidden,     // (default) no tags
	}
)

// CheckpointDecOptions returns the decoding options compatible with the RootSigner
// With these options the sign is always retained
// The options align with the cbor defaults, except for the handling of unsigned integers.
func CheckpointDecOptions() cbor.DecOptions {
	return decOptions
}

// CheckpointEncOptions returns the decoding options compatible with the RootSigner
// These options align with the cbor defaults
func CheckpointEncOptions() cbor.EncOptions {
	return encOptions
}

func newCheckpointDecOptions() []commoncose.SignOption {
	return []commoncose.SignOption{commoncose.WithDecOptions(decOptions)}
}
