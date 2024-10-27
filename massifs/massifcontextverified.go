package massifs

import (
	"context"
	"crypto"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/datatrails/go-datatrails-common/cose"
	"github.com/datatrails/go-datatrails-merklelog/mmr"
)

// Support for an extension to the MassifContext type that provides for getting
// massif and verifying it at the same time. The primary caller interface is
// GetVerifiedMassif, and all the other methods are in support of that.  Where
// possible, and useful, the methods are made available directly on the
// MassifContext type itself

var (
	ErrStateSizeBeforeMassifStart = errors.New("the massif index in the mmr state must at least cover the start of the massif")
	ErrStateSizeExceedsData       = errors.New("There is insufficient data in the massif context to generate a consistency proof against the provided state")
	ErrSealGetterNotProvided      = errors.New("a seal getter was required but not provided")
	ErrCBORCodecNotProvided       = errors.New("a CBOR codec was required but not provided")
	ErrSealNotFound               = errors.New("seal not found")
	ErrSealVerifyFailed           = errors.New("the seal signature verification failed")
	ErrGeneratingConsistencyProof = errors.New("error while  creating a consistency proof")
	ErrConsistencyProofCheck      = errors.New("verification error while checking a consistency proof")
	ErrInconsistentState          = errors.New("verification failed for a consistency proof")
	ErrRemoteSealKeyMatchFailed   = errors.New("the provided public key did not match the remote sealing key")
	ErrTenantIdUnknown            = errors.New("the method requires that the tenant ientity is known on the context")
	ErrTenantIdInconsistent       = errors.New("the tenant identity on the context does not match the tenant identity provided")
)

// VerifiedContext describes a verified massif context
//
// Methods which both read a massif and then require that the massif's associated
// seal can be verified, against the read data, should return a VerifiedContext.
type VerifiedContext struct {
	MassifContext

	// The signed message that was used to verify the massif data. Verification
	// will use the public key from this message. The verification method allows
	// the caller to provide the public key they expect, based on having
	// obtained it from a store they trust. Where the expected public key has
	// been provided it is required to match the key found on the seal from the
	// store (which may be local or remote).
	Sign1Message cose.CoseSign1Message
	// MMRState describes the sealed (confirmed) range of the massif. For a verified massif
	// context it is guaranteed to refer to the portion of the log identified by
	// massifIndex, but the committed data may extend past the data confirmed by
	// the seal.
	MMRState MMRState

	// ConsistentRoots is the result of verifying the entire range of the massif
	// context data against the seal state for the massif. If a previously
	// trusted state was provided when verification was performed, this state is
	// also consistent with that.  When configured to use "bagged" peaks for
	// verification purposes, this will be the single bagged root of the mmr up to the
	// end of the data.  Otherwise, it will be the accumulator peaks.
	ConsistentRoots [][]byte
}

// checkedVerifiedContextOptions checks the options provided satisfy the common requirements of the reader methods
func checkedVerifiedContextOptions(baseOpts ReaderOptions, opts ...ReaderOption) (ReaderOptions, error) {
	options := ReaderOptionsCopy(baseOpts)
	for _, o := range opts {
		o(&options)
	}

	if options.sealGetter == nil {
		return ReaderOptions{}, ErrSealGetterNotProvided
	}

	if options.codec == nil {
		return ReaderOptions{}, ErrCBORCodecNotProvided
	}
	return options, nil
}

// GetHeadVerifiedContext gets the massif and its seal and then verifies the massif
// data against the seal. If the caller provides the expected public key, the
// public key on the seal is required to match
func (mr *MassifReader) GetHeadVerifiedContext(
	ctx context.Context, tenantIdentity string,
	opts ...ReaderOption,
) (*VerifiedContext, error) {

	options, err := checkedVerifiedContextOptions(mr.opts, opts...)
	if err != nil {
		return nil, err
	}

	mc, err := mr.GetHeadMassif(ctx, tenantIdentity, opts...)
	if err != nil {
		return nil, err
	}

	return mc.verifyContext(ctx, options)
}

// GetVerifiedContext gets the massif and its seal and then verifies the massif
// data against the seal. If the caller provides the expected public key, the
// public key on the seal is required to match
func (mr *MassifReader) GetVerifiedContext(
	ctx context.Context, tenantIdentity string, massifIndex uint64,
	opts ...ReaderOption,
) (*VerifiedContext, error) {

	options, err := checkedVerifiedContextOptions(mr.opts, opts...)
	if err != nil {
		return nil, err
	}

	mc, err := mr.GetMassif(ctx, tenantIdentity, massifIndex, opts...)
	if err != nil {
		return nil, err
	}

	return mc.verifyContext(ctx, options)
}

// VerifyContext verifies an arbitrary context and returns a verified context if this succeeds.
// This performs the same checks as GetVerifiedContext
func (mr *MassifReader) VerifyContext(
	ctx context.Context, mc MassifContext,
	opts ...ReaderOption,
) (*VerifiedContext, error) {
	options, err := checkedVerifiedContextOptions(mr.opts, opts...)
	if err != nil {
		return nil, err
	}

	return mc.verifyContext(ctx, options)
}

// VerifyContext verifies the context and returns a verified context if this succeeds.
func (mc *MassifContext) VerifyContext(
	ctx context.Context,
	opts ...ReaderOption,
) (*VerifiedContext, error) {

	options, err := checkedVerifiedContextOptions(ReaderOptions{}, opts...)
	if err != nil {
		return nil, err
	}
	return mc.verifyContext(ctx, options)
}

// verifyContext verifies the log data in the context is consistent with its seal
// optionally also checks consistency against a provided state from a trusted source
// Returns:
//   - a VerifiedContext which references the dynamically allocated aspects of this context
func (mc *MassifContext) verifyContext(
	ctx context.Context, options ReaderOptions,
) (*VerifiedContext, error) {

	var ok bool
	var err error
	var peaksB [][]byte

	// This checks that any un-committed data is consistent with the latest seal available for the massif

	msg, state, err := options.sealGetter.GetSignedRoot(ctx, mc.TenantIdentity, mc.Start.MassifIndex)
	if err != nil {
		if IsBlobNotFound(err) {
			return nil, fmt.Errorf(
				"%w: failed to get seal for massif %d for tenant %s: %v",
				ErrSealNotFound, mc.Start.MassifIndex, mc.TenantIdentity, WrapBlobNotFound(err))
		}
		return nil, err
	}

	// get the peaks from the local store, we are checking the store against the
	// latest additions. as we verify the signature below, any changes to the
	// store will be caught.
	state.Peaks, err = mmr.PeakHashes(mc, state.MMRSize-1)
	if err != nil {
		return nil, err
	}

	// NOTICE: The verification uses the public key that is provided on the
	// message.  If the caller wants to ensure the massif is signed by the
	// expected key then they must obtain a copy of the public key from a source
	// they trust and supply it as an option.
	pubKeyProvider := cose.NewCWTPublicKeyProvider(msg)

	if options.trustedSealerPubKey != nil {
		var remotePub crypto.PublicKey
		remotePub, _, err = pubKeyProvider.PublicKey()
		if err != nil {
			return nil, err
		}
		if !options.trustedSealerPubKey.Equal(remotePub) {
			return nil, ErrRemoteSealKeyMatchFailed
		}
	}

	// Ensure the peaks we read from the store are the ones that were signed.
	// Otherwise we can get caught out by the store tampered after the seal was
	// created. Of course the seal itself could have been replaced, but at that
	// point the only defense is an indpendent replica.
	err = VerifySignedCheckPoint(
		*options.codec, pubKeyProvider, msg, state, nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: failed to verify seal for massif %d for tenant %s: %v",
			ErrSealVerifyFailed, mc.Start.MassifIndex, mc.TenantIdentity, err)
	}

	// This verifies the peaks read from mmrSizeA are consistent with mmrSizeB.
	ok, peaksB, err = mmr.CheckConsistency(
		mc, sha256.New(), state.MMRSize, mc.RangeCount(), state.Peaks)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: error verifying accumulator state from massif %d for tenant %s",
			err, mc.Start.MassifIndex, mc.TenantIdentity)
	}
	if !ok {
		// We don't expect false without error, but we
		return nil, fmt.Errorf("%w: failed to verify accumulator state massif %d for tenant %s",
			mmr.ErrConsistencyCheck, mc.Start.MassifIndex, mc.TenantIdentity)
	}

	// If the caller has provided a trusted base state, also verify against
	// that. Typically this is used for 3d party verification, the 3rd party has
	// saved a previously verified state in a local store, and they want to
	// check the remote log is consistent with the log portion they have locally
	// before replicating the new data.
	if options.trustedBaseState != nil {

		ok, _, err = mmr.CheckConsistency(
			mc, sha256.New(),
			options.trustedBaseState.MMRSize,
			mc.RangeCount(),
			options.trustedBaseState.Peaks)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf(
				"%w: the accumulator produced for the trusted base state doesn't match the root produced for the seal state fetched from the log",
				mmr.ErrConsistencyCheck)
		}
	}

	return &VerifiedContext{
		MassifContext:   *mc,
		Sign1Message:    *msg,
		MMRState:        state,
		ConsistentRoots: peaksB,
	}, nil
}
