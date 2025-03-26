package massifs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/datatrails/go-datatrails-common/azblob"
	commoncbor "github.com/datatrails/go-datatrails-common/cbor"
	"github.com/fxamacker/cbor/v2"

	commoncose "github.com/datatrails/go-datatrails-common/cose"
	"github.com/datatrails/go-datatrails-common/logger"
	"github.com/datatrails/go-datatrails-merklelog/mmr"
)

// MMRIVER COSE Receipts to accompany our COSE MMRIVER seals

type MMRiverInclusionProof struct {
	Index         uint64   `cbor:"1,keyasint"`
	InclusionPath [][]byte `cbor:"2,keyasint"`
}

type MMRiverConsistencyProof struct {
	TreeSize1        uint64   `cbor:"1,keyasint"`
	TreeSize2        uint64   `cbor:"2,keyasint"`
	ConsistencyPaths [][]byte `cbor:"3,keyasint"`
	RightPeaks       [][]byte `cbor:"4,keyasint"`
}

type MMRiverVerifiableProofs struct {
	InclusionProofs   []MMRiverInclusionProof   `cbor:"-1,keyasint,omitempty"`
	ConsistencyProofs []MMRiverConsistencyProof `cbor:"-2,keyasint,omitempty"`
}

// MMRiverInclusionProofHeader provides for encoding, and defered decoding, of
// COSE_Sign1 message headers for MMRIVER receipts
type MMRiverVerifiableProofsHeader struct {
	VerifiableProofs MMRiverVerifiableProofs `cbor:"396,keyasint"`
}

// VerifySignedInclusionReceipts verifies a signed COSE receipt encoded according to the MMRIVER VDS
// on success the produced root is returned.
// Signature verification failure is not an error, but the returned root will be nil and the result will be false.
// All other unexpected issues are returned as errors, with a false result and nil root.
// Note that MMRIVER receipts allow for multiple inclusion proofs to be attached to the receipt.
// This function returns true only if ALL receipts verify
//
// The candidates array provides the *candidate* values. Once verified, we can call them node values (or leaves),
// Note that any node value in the log may be proven by a receipt, not just leaves.
func VerifySignedInclusionReceipts(
	ctx context.Context,
	receipt *commoncose.CoseSign1Message,
	candidates [][]byte,
) (bool, []byte, error) {

	var err error

	// ignore any existing payload
	receipt.Payload = nil

	// We must return false if there are no candidates
	if len(candidates) == 0 {
		return false, nil, fmt.Errorf("no candidates provided")
	}

	var header MMRiverVerifiableProofsHeader
	err = cbor.Unmarshal(receipt.Headers.RawUnprotected, &header)
	if err != nil {
		return false, nil, fmt.Errorf("MMRIVER receipt proofs malformed")
	}
	verifiableProofs := header.VerifiableProofs
	if len(verifiableProofs.InclusionProofs) == 0 {
		return false, nil, fmt.Errorf("MMRIVER receipt inclusion proofs not present")
	}

	// permit *fewer* candidates than proofs, but not more
	if len(candidates) > len(verifiableProofs.InclusionProofs) {
		return false, nil, fmt.Errorf("MMRIVER receipt more candidates than proofs")
	}

	var proof MMRiverInclusionProof

	proof = verifiableProofs.InclusionProofs[0]
	receipt.Payload = mmr.IncludedRoot(
		sha256.New(),
		proof.Index, candidates[0],
		proof.InclusionPath)

	err = receipt.VerifyWithCWTPublicKey(nil)
	if err != nil {
		return false, nil, fmt.Errorf(
			"MMRIVER receipt VERIFY FAILED for: mmrIndex %d, candidate %d, err %v", proof.Index, 0, err)
	}
	// verify the first proof then just compare the produced roots

	for i := 1; i < len(verifiableProofs.InclusionProofs); i++ {

		proof = verifiableProofs.InclusionProofs[i]
		proven := mmr.IncludedRoot(sha256.New(), proof.Index, candidates[i], proof.InclusionPath)
		if bytes.Compare(receipt.Payload, proven) != 0 {
			return false, nil, fmt.Errorf(
				"MMRIVER receipt VERIFY FAILED for: mmrIndex %d, candidate %d, err %v", proof.Index, i, err)
		}
	}
	return true, receipt.Payload, nil
}

// VerifySignedInclusionReceipt verifies a reciept comprised of a single inclusion proof
// If there are 0 or more than 1 candidates, the result will be false and an error will be returned
func VerifySignedInclusionReceipt(
	ctx context.Context,
	receipt *commoncose.CoseSign1Message,
	candidate []byte,
) (bool, []byte, error) {

	ok, root, err := VerifySignedInclusionReceipts(ctx, receipt, [][]byte{candidate})
	if err != nil {
		return false, nil, err
	}
	if !ok {
		return false, nil, nil
	}
	return true, root, nil
}

type verifiedContextGetter interface { //nolint:iface
	GetVerifiedContext(
		ctx context.Context, tenantIdentity string, massifIndex uint64,
		opts ...ReaderOption,
	) (*VerifiedContext, error)
}

// NewReceipt returns a COSE receipt for the given tenantIdentity and mmrIndex
func NewReceipt(
	ctx context.Context,
	massifHeight uint8,
	tenantIdentity string, mmrIndex uint64,
	getter verifiedContextGetter,
) (*commoncose.CoseSign1Message, error) {

	log := logger.Sugar.FromContext(ctx)
	defer log.Close()
	massifIndex := uint32(MassifIndexFromMMRIndex(massifHeight, mmrIndex))

	verified, err := getter.GetVerifiedContext(ctx, tenantIdentity, uint64(massifIndex))
	if err != nil {
		return nil, fmt.Errorf(
			"%w: failed to get verified context %d for %s", err, massifIndex, tenantIdentity)
	}

	msg, state := verified.Sign1Message, verified.MMRState

	proof, err := mmr.InclusionProof(&verified.MassifContext, state.MMRSize-1, mmrIndex)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to generating inclusion proof: %d in MMR(%d), %v",
			mmrIndex, verified.MMRState.MMRSize, err)
	}

	peakIndex := mmr.PeakIndex(mmr.LeafCount(state.MMRSize), len(proof))

	// NOTE: The old-accumulator compatibility property, from
	// https://eprint.iacr.org/2015/718.pdf, along with the COSE protected &
	// unprotected buckets, is why we can just pre sign the receipts.
	// As long as the receipt consumer is convinced of the logs consistency (not split view),
	// it does not matter which accumulator state the receipt is signed against.

	var peaksHeader MMRStateReceipts
	err = cbor.Unmarshal(msg.Headers.RawUnprotected, &peaksHeader)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: failed decoding peaks header: for tenant %s, seal %d", err, tenantIdentity, massifIndex)
	}
	if peakIndex >= len(peaksHeader.PeakReceipts) {
		return nil, fmt.Errorf(
			"%w: peaks header containes to few peak receipts: for tenant %s, seal %d", err, tenantIdentity, massifIndex)
	}

	// This is an array of marshaled COSE_Sign1's
	receiptMsg := peaksHeader.PeakReceipts[peakIndex]
	signed, err := commoncose.NewCoseSign1MessageFromCBOR(
		receiptMsg, commoncose.WithDecOptions(CheckpointDecOptions()))
	if err != nil {
		return nil, fmt.Errorf(
			"%w: failed to decode pre-signed receipt for: %d in MMR(%d)",
			err, mmrIndex, state.MMRSize)
	}

	// signed.Headers.RawProtected = nil
	signed.Headers.RawUnprotected = nil

	verifiableProofs := MMRiverVerifiableProofs{
		InclusionProofs: []MMRiverInclusionProof{{
			Index:         mmrIndex,
			InclusionPath: proof}},
	}

	signed.Headers.Unprotected[VDSCoseReceiptProofsTag] = verifiableProofs

	return signed, nil
}

type ReceiptBuilder struct {
	log          logger.Logger
	massifReader MassifReader
	cborCodec    commoncbor.CBORCodec
	massifHeight uint8
}

// newReceiptBuilder creates a new receiptBuilder configured with all the necessary readers and information required to build a receipt
// Note that errors are logged assuming the calling context is retrieving a receipt,
// and that all returned errors are StatusErrors that can be returned to the client or nil
func NewReceiptBuilder(log logger.Logger, reader azblob.Reader, massifHeight uint8) (ReceiptBuilder, error) {

	var err error

	b := ReceiptBuilder{
		log:          log,
		massifHeight: massifHeight,
	}

	if b.cborCodec, err = NewRootSignerCodec(); err != nil {
		return ReceiptBuilder{}, err
	}
	b.massifHeight = massifHeight
	b.massifReader = NewMassifReader(log, reader)
	sealReader := NewSignedRootReader(log, reader, b.cborCodec)
	b.massifReader = NewMassifReader(log, reader, WithSealGetter(&sealReader))

	return b, nil
}

func (b *ReceiptBuilder) BuildReceipt(
	ctx context.Context, tenantIdentity string, mmrIndex uint64,
) (*commoncose.CoseSign1Message, error) {

	log := b.log.FromContext(ctx)
	defer log.Close()

	return NewReceipt(ctx, b.massifHeight, tenantIdentity, mmrIndex, &b.massifReader)
}
