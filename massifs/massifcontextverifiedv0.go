package massifs

import (
	"crypto/sha256"
	"fmt"

	"github.com/datatrails/go-datatrails-common/cose"
	"github.com/datatrails/go-datatrails-merklelog/mmr"
)

// verifyContextV0 reads and verifies the V0 seal, which will remain important
// for replicas for a while.
func (mc *MassifContext) verifyContextV0(
	msg *cose.CoseSign1Message, state MMRState, options ReaderOptions,
) (*VerifiedContext, error) {

	var ok bool
	var err error
	var rootB []byte

	if state.Version != int(MMRStateVersion0) {
		return nil, fmt.Errorf("unsupported MMR state version %d", state.Version)
	}

	state.LegacySealRoot, err = mmr.GetRoot(state.MMRSize, mc, sha256.New())
	if err != nil {
		return nil, err
	}

	pubKeyProvider, err := mc.sealPublicKeyProvider(msg, options)
	if err != nil {
		return nil, err
	}

	err = VerifySignedCheckPoint(
		*options.codec, pubKeyProvider, msg, state, nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: failed to verify seal for massif %d for tenant %s: %v",
			ErrSealVerifyFailed, mc.Start.MassifIndex, mc.TenantIdentity, err)
	}
	cp, err := mmr.IndexConsistencyProofBagged(
		state.MMRSize, mc.RangeCount(), mc, sha256.New())
	if err != nil {
		return nil, fmt.Errorf(
			"%w: error creating bagged consistency proof from %d for massif %d, for tenant %s",
			err, state.MMRSize, mc.Start.MassifIndex, mc.TenantIdentity)
	}

	ok, rootB, err = mmr.CheckConsistencyBagged(mc, sha256.New(), cp, state.LegacySealRoot)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: error verifying bagged consistency proof from %d for massif %d, for tenant %s",
			err, state.MMRSize, mc.Start.MassifIndex, mc.TenantIdentity)
	}
	if !ok {
		// We don't expect false without error, but we
		return nil, fmt.Errorf("%w: failed to verify bagged state for massif %d for tenant %s",
			mmr.ErrConsistencyCheck, mc.Start.MassifIndex, mc.TenantIdentity)
	}
	// If the caller has provided a trusted base state, also verify against
	// that. Typically this is used for 3d party verification, the 3rd party has
	// saved a previously verified state in a local store, and they want to
	// check the remote log is consistent with the log portion they have locally
	// before replicating the new data.
	if options.trustedBaseState != nil {

		cp, err := mmr.IndexConsistencyProofBagged(
			state.MMRSize, mc.RangeCount(), mc, sha256.New())
		if err != nil {
			return nil, fmt.Errorf(
				"%w: error checking consistency with trusted base state from %d for tenant %s",
				err, options.trustedBaseState.MMRSize, mc.TenantIdentity)
		}

		ok, _, err = mmr.CheckConsistencyBagged(mc, sha256.New(), cp, options.trustedBaseState.LegacySealRoot)
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
		ConsistentRoots: [][]byte{rootB},
	}, nil
}
