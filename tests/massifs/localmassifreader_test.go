//go:build integration && azurite

package massifs

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/datatrails/go-datatrails-common/azkeys"
	"github.com/datatrails/go-datatrails-common/cbor"
	"github.com/datatrails/go-datatrails-common/cose"
	"github.com/datatrails/go-datatrails-common/logger"
	"github.com/datatrails/go-datatrails-merklelog/massifs"
	"github.com/datatrails/go-datatrails-merklelog/massifs/mocks"
	"github.com/datatrails/go-datatrails-merklelog/mmr"
	"github.com/datatrails/go-datatrails-merklelog/mmrtesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	gocose "github.com/veraison/go-cose"
)

// TestLocalMassifReaderGetVerifiedContext ensures the conistency checks are
// performed according to the avialable data and the provide options
//
// The major log verification scenarios tested are:
//  1. The remote massif data has been tampered to include, modify or exclude a leaf.
//  2. The remote massif data has been extended inconsistently with respect to ealier un-tampered data.
//  3. The remote massif data and the latest remote seal have been tampered, but a previously "known good" seal for the massif is available locally.
//
// The signing key verification scenarios tested are:
//  1. It can be absent
//  2. It can be present and match the sealing public key
//  3. It can be present and NOT match the sealing public key
func TestLocalMassifReaderGetVerifiedContext(t *testing.T) {
	logger.New("TestLocalMassifReaderGetVerifiedContext")
	defer logger.OnExit()

	tc := newLocalMassifReaderTestContext(
		t, logger.Sugar, "TestLocalMassifReaderGetVerifiedContext")

	tenantId0 := tc.g.NewTenantIdentity()
	tenantId1SealBehindLog := tc.g.NewTenantIdentity()
	tenantId2TamperedLogUpdate := tc.g.NewTenantIdentity()
	tenantId3InconsistentLogUpdate := tc.g.NewTenantIdentity()
	tenantId4RemoteInconsistentWithTrustedSeal := tc.g.NewTenantIdentity()
	tenantId5TrustedPublicKeyMismatch := tc.g.NewTenantIdentity()

	allTenants := []string{tenantId0, tenantId1SealBehindLog, tenantId2TamperedLogUpdate, tenantId3InconsistentLogUpdate, tenantId4RemoteInconsistentWithTrustedSeal, tenantId5TrustedPublicKeyMismatch}

	massifHeight := uint8(8)
	tc.CreateLog(tenantId0, massifHeight, 3*(1<<massifHeight)+0)
	tc.CreateLog(tenantId1SealBehindLog, massifHeight, 3*(1<<massifHeight)+1)
	tc.CreateLog(tenantId2TamperedLogUpdate, massifHeight, 3*(1<<massifHeight)+2)
	tc.CreateLog(tenantId3InconsistentLogUpdate, massifHeight, 3*(1<<massifHeight)+3)
	tc.CreateLog(tenantId4RemoteInconsistentWithTrustedSeal, massifHeight, 3*(1<<massifHeight)+4)
	tc.CreateLog(tenantId5TrustedPublicKeyMismatch, massifHeight, 3*(1<<massifHeight)+5)

	// sizeBeforeLeaves returns the size of the massif before the leaves provded number of leaves were added
	sizeBeforeLeaves := func(mc *massifs.MassifContext, leavesBefore uint64) uint64 {
		mmrSize := mc.RangeCount()
		leafCount := mmr.LeafCount(mmrSize)
		oldLeafCount := leafCount - leavesBefore
		mmrSizeOld := mmr.FirstMMRSize(mmr.TreeIndex(oldLeafCount - 1))
		return mmrSizeOld
	}

	findMassif := func(identifier string, massifIndex uint64) (*massifs.MassifContext, error) {
		for _, tenantId := range allTenants {
			if !strings.Contains(identifier, tenantId) {
				continue
			}
			mc, err := tc.azuriteReader.GetMassif(context.TODO(), tenantId, massifIndex)
			if err != nil {
				return nil, err
			}
			return &mc, nil
		}
		return nil, massifs.ErrMassifNotFound
	}

	tamperNode := func(mc *massifs.MassifContext, mmrIndex uint64) {
		require.GreaterOrEqual(t, mmrIndex, mc.Start.FirstIndex)
		i := mmrIndex - mc.Start.FirstIndex
		logData := mc.Data[mc.LogStart():]
		tamperedBytes := []byte{0x0D, 0x0E, 0x0A, 0x0D, 0x0B, 0x0E, 0x0E, 0x0F}
		copy(logData[i*massifs.LogEntryBytes:i*massifs.LogEntryBytes+8], tamperedBytes)
	}

	seal := func(
		mc *massifs.MassifContext, mmrSize uint64, tenantIdentity string, massifIndex uint32,
	) (*cose.CoseSign1Message, massifs.MMRState, error) {
		root, err := mmr.GetRoot(mmrSize, mc, sha256.New())
		if err != nil {
			return nil, massifs.MMRState{}, err
		}
		return tc.SignedState(tenantIdentity, uint64(massifIndex), massifs.MMRState{
			MMRSize: mmrSize, Root: root,
		})
	}

	sg := *mocks.NewSealGetter(t)
	sg.On("GetSignedRoot", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		func(
			ctx context.Context, tenantIdentity string, massifIndex uint32,
			opts ...massifs.ReaderOption,
		) (*cose.CoseSign1Message, massifs.MMRState, error) {
			mc, err := findMassif(tenantIdentity, uint64(massifIndex))
			if err != nil {
				return nil, massifs.MMRState{}, err
			}
			switch tenantIdentity {
			case tenantId1SealBehindLog:
				// Common case: Return a seal that omits the last few leaves. In this case GetVerifiedContext should
				// return the root *inclusive* of the additional leaves, having verified the seal over only the original leaves.
				mmrSize := mc.RangeCount()
				leafCount := mmr.LeafCount(mmrSize)
				sealedLeafCount := leafCount - 8
				mmrSizeOld := mmr.FirstMMRSize(mmr.TreeIndex(sealedLeafCount - 1))
				require.GreaterOrEqual(t, mmrSizeOld, mc.Start.FirstIndex)

				return seal(mc, mmrSizeOld, tenantIdentity, massifIndex)
			case tenantId2TamperedLogUpdate:

				// We are simulating a situation where the locally available
				// root seal for the *earlier* massif data is correct, but the
				// updated (extended) log obtained from the remote (datatrails)
				// has been tampered with. To simulate this, we first obtain
				// the seal from the un-tampered log. Then, we tamper with the
				// log data. The verification will fail because the tampered log
				// will produce a different root.

				// Note that every time the GetSignedRoot mock or the ReadMassif
				// mock is called, the data is read fresh from the azurite
				// store.

				// Tampering a log requires updating all nodes after the
				// tampered node to maintain verifiability. In the case of a
				// delete, inclusion proofs for subsequent nodes would become
				// invalid. Therefore, in this simulation, we can simulate a
				// tampered *rebuilt* log, without actually re-building it, by
				// changing a peak node directly.

				// Detecting a gratuitously tampered leaf, where the tree is not
				// re-built, is the reason why third-party auditors are included
				// in the security model. In such cases, the seal would remain
				// unaffected, but nothing in the tampered log would verify against it.

				// It is important to note that all tamper scenarios still require
				// the attacker to have access to the signing key.

				// note; we don't specifically need to work with a mid massif
				// state here, we just do so to show this works for an arbitrary
				// seal point, and for alignment with consistency check tests

				mmrSizeOld := sizeBeforeLeaves(mc, 8)
				require.GreaterOrEqual(t, mmrSizeOld, mc.Start.FirstIndex)

				// Get the seal before applying the tamper
				msg, state, err := seal(mc, mmrSizeOld, tenantIdentity, massifIndex)
				if err != nil {
					return nil, massifs.MMRState{}, err
				}

				root, _ := mmr.GetRoot(state.MMRSize, mc, sha256.New())
				peaks := mmr.Peaks(mmrSizeOld)
				// Remember, the peaks are *positions*

				// Note: we take the *last* peak, because it corresponds to the
				// most recent log entries, but tampering any peak will cause
				// the verification to fail to fail
				tamperNode(mc, peaks[len(peaks)-1]-1)

				root2, _ := mmr.GetRoot(state.MMRSize, mc, sha256.New())

				assert.NotEqual(t, root, root2, "tamper did not change the root")

				// Now we can return the seal
				return msg, state, nil

			case tenantId3InconsistentLogUpdate:

				// In this case, the log is un-tampered, up to the seal, but the additions after the seal are inconsistent.

				// tamper *after* the seal
				mmrSizeOld := sizeBeforeLeaves(mc, 8)
				require.GreaterOrEqual(t, mmrSizeOld, mc.Start.FirstIndex)

				// Get the seal before applying the tamper
				msg, state, err := seal(mc, mmrSizeOld, tenantIdentity, massifIndex)
				if err != nil {
					return nil, massifs.MMRState{}, err
				}

				// this time, tamper a peak after the seal, this simulates the
				// case where the extension is inconsistent with the seal.
				peaks := mmr.Peaks(mc.RangeCount())

				// Note: we take the *last* peak, because it corresponds to the
				// most recent log entries. In this case we want the fresh
				// additions to the log to be inconsistent with the seal. Until
				// enough new entries are added, those new entries are only
				// dependent on the smallest sealed peak.

				// Remember, the peaks are *positions*
				tamperNode(mc, peaks[len(peaks)-1]-1)

				// Now we can return the seal
				return msg, state, nil

			default:
				// Common case: the seal is the full extent of the massif
				return seal(mc, mc.RangeCount(), tenantIdentity, massifIndex)
			}
		})

	dc := mocks.NewDirCache(t)
	dc.On("Options", mock.Anything).Return(
		func() massifs.DirCacheOptions {
			return massifs.NewLogDirCacheOptions(
				massifs.ReaderOptions{},
				massifs.WithReaderOption(massifs.WithSealGetter(&sg)),
				massifs.WithReaderOption(massifs.WithCBORCodec(tc.rootSignerCodec)),
			)
		})
	dc.On("GetEntry", mock.Anything).Return(
		func(directory string) (*massifs.LogDirCacheEntry, bool) {
			return nil, false
		})
	dc.On("ResolveDirectory", mock.Anything).Return(
		func(directory string) (string, error) {
			return directory, nil
		})
	dc.On("ReadMassif", mock.Anything, mock.Anything).Return(
		func(directory string, massifIndex uint64) (*massifs.MassifContext, error) {

			mc, err := findMassif(directory, massifIndex)
			if err != nil {
				return nil, err
			}
			switch directory {
			case tenantId2TamperedLogUpdate:

				// For the seal verification check, we ensure that the seal is
				// generated against the un tampered data in the GetSignedRoot
				// mock. Here, we ensure that all other observers see only the
				// tampered data.

				mmrSizeOld := sizeBeforeLeaves(mc, 8)
				require.GreaterOrEqual(t, mmrSizeOld, mc.Start.FirstIndex)
				peaks := mmr.Peaks(mmrSizeOld)
				// remember, the peaks are *positions*
				tamperNode(mc, peaks[len(peaks)-1]-1)

			case tenantId3InconsistentLogUpdate:
				// tamper *after* the seal
				// this time, tamper a peak after the seal, this simulates the
				// case where the extension is inconsistent with the seal.
				peaks := mmr.Peaks(mc.RangeCount())
				// Remember, the peaks are *positions*
				tamperNode(mc, peaks[len(peaks)-1]-1)

			default:
			}
			return mc, nil
		})

	// To provoke the case where the local, trusted, seal is inconsistent with
	// the remote seal & log, we play a bit of a trick. We get a seal for a
	// *tampered* log, then later, ALL the legit log data will otherwise verify
	// but will fail against the "trusted good seal". It is precisesly the
	// opposite of what we are protecting against in the real world, but it is
	// equivelent from a test perspective.

	mc, err := findMassif(tenantId4RemoteInconsistentWithTrustedSeal, 0)
	require.NoError(t, err)
	mmrSizeOld := sizeBeforeLeaves(mc, 8)
	require.GreaterOrEqual(t, mmrSizeOld, mc.Start.FirstIndex)
	peaks := mmr.Peaks(mmrSizeOld)
	// remember, the peaks are *positions*
	tamperNode(mc, peaks[len(peaks)-1]-1)

	// We  call this a fake good state because its actually tampered, and the
	// log is "good", but it has the same effect from a verification
	// perspective.
	_, fakeGoodState, err := seal(mc, mmrSizeOld, tenantId4RemoteInconsistentWithTrustedSeal, 0)
	require.NoError(t, err)

	fakeECKey := mustGenerateECKey(t, elliptic.P256())

	type logStates struct {
		mmrSize uint64
	}

	type args struct {
		tenantIdentity string
		// The massifIndex is used to identify the test case's desired results to the mock implementations above.
		massifIndex uint64
	}
	tests := []struct {
		name          string
		callOpts      []massifs.ReaderOption
		args          args
		wantErr       error
		wantErrPrefix string
	}{
		// provide an invalid public signing key, this simulates a remote log being signed by a different key than the verifier expects
		// {name: "invalid public seal key", args: args{tenantIdentity: tenantId3InconsistentLogUpdate, massifIndex: 0}, wantErr: massifs.ErrRemoteSealKeyMatchFailed},
		{
			name:     "valid public seal key",
			callOpts: []massifs.ReaderOption{massifs.WithTrustedSealerPub(&tc.key.PublicKey)},
			args:     args{tenantIdentity: tenantId5TrustedPublicKeyMismatch, massifIndex: 0},
			wantErr:  nil,
		},

		{
			name:     "invalid public seal key",
			callOpts: []massifs.ReaderOption{massifs.WithTrustedSealerPub(&fakeECKey.PublicKey)},
			args:     args{tenantIdentity: tenantId5TrustedPublicKeyMismatch, massifIndex: 0},
			wantErr:  massifs.ErrRemoteSealKeyMatchFailed,
		},

		{
			name:     "local seal inconsistent with remote log",
			callOpts: []massifs.ReaderOption{massifs.WithTrustedBaseState(fakeGoodState)}, args: args{tenantIdentity: tenantId4RemoteInconsistentWithTrustedSeal, massifIndex: 0},
			wantErr: massifs.ErrInconsistentState,
		},
		// see the GetSignedRoot mock above for the rational behind tampering only a peak
		{name: "tamper after seal", args: args{tenantIdentity: tenantId3InconsistentLogUpdate, massifIndex: 0}, wantErr: massifs.ErrInconsistentState},
		{name: "seal peak tamper", args: args{tenantIdentity: tenantId2TamperedLogUpdate, massifIndex: 0}, wantErr: gocose.ErrVerification},
		{name: "seal shorter than massif", args: args{tenantIdentity: tenantId1SealBehindLog, massifIndex: 0}},
		{name: "happy path", args: args{tenantIdentity: tenantId0, massifIndex: 0}},
	}
	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {

				reader, err := massifs.NewLocalReader(logger.Sugar, dc)
				assert.NoError(t, err)
				_, err = reader.GetVerifiedContext(
					context.TODO(),
					tt.args.tenantIdentity,
					tt.args.massifIndex,
					append(tt.callOpts, massifs.WithSealGetter(&sg))...)

				if tt.wantErr == nil {
					assert.Nil(t, err, "unexpected error")
				} else if tt.wantErr != nil {
					assert.NotNil(t, err, "expected error got nil")
					assert.ErrorIs(t, err, tt.wantErr)
				} else if tt.wantErrPrefix != "" {
					assert.NotNil(t, err, "expected error got nil")
					assert.True(t, strings.HasPrefix(err.Error(), tt.wantErrPrefix))
				}
				if tt.wantErr == nil || tt.wantErrPrefix == "" {
					return
				}
			},
		)
	}

}

type testLocalMassifReaderContext struct {
	testSignerContext
	azuriteContext   mmrtesting.TestContext
	mmrtestingConfig mmrtesting.TestConfig
	committerConfig  massifs.TestCommitterConfig

	g mmrtesting.TestGenerator
	// We use a regular massif reader attached to azurite to test the local massif reader.
	azuriteReader massifs.MassifReader
}

func (c *testLocalMassifReaderContext) CreateLog(tenantIdentity string, massifHeight uint8, mmrSize uint64) {

	// clear out any previous log
	c.azuriteContext.DeleteBlobsByPrefix(massifs.TenantMassifPrefix(tenantIdentity))

	committer, err := massifs.NewTestMinimalCommitter(massifs.TestCommitterConfig{
		CommitmentEpoch: 1,
		MassifHeight:    massifHeight,
	}, c.azuriteContext, c.g, massifs.MMRTestingGenerateNumberedLeaf)
	require.NoError(c.azuriteContext.T, err)

	massifSize := mmr.HeightSize(uint64(massifHeight)) + 1
	massifCount := mmrSize / massifSize
	if massifCount == 0 {
		c.azuriteContext.T.FailNow()
	}
	lastSize := mmrSize - massifCount*massifSize

	base := uint64(0)
	for i := 0; i < int(massifCount)-1; i++ {
		err := committer.AddLeaves(context.TODO(), tenantIdentity, base, massifSize)
		require.NoError(c.azuriteContext.T, err)
	}
	if lastSize > 0 {
		err := committer.AddLeaves(context.TODO(), tenantIdentity, base, massifSize)
		require.NoError(c.azuriteContext.T, err)
	}
}

func newLocalMassifReaderTestContext(
	t *testing.T, log logger.Logger, testLabelPrefix string) testLocalMassifReaderContext {
	cfg := mmrtesting.TestConfig{
		StartTimeMS: (1698342521) * 1000, EventRate: 500,
		TestLabelPrefix: testLabelPrefix,
		TenantIdentity:  "",
		Container:       strings.ReplaceAll(strings.ToLower(testLabelPrefix), "_", ""),
	}

	tc := mmrtesting.NewTestContext(t, cfg)

	g := mmrtesting.NewTestGenerator(
		t, cfg.StartTimeMS/1000,
		mmrtesting.TestGeneratorConfig{
			StartTimeMS:     cfg.StartTimeMS,
			EventRate:       cfg.EventRate,
			TenantIdentity:  cfg.TenantIdentity,
			TestLabelPrefix: cfg.TestLabelPrefix,
		},
		massifs.MMRTestingGenerateNumberedLeaf,
	)

	signer := newTestSignerContext(t, testLabelPrefix)
	return testLocalMassifReaderContext{
		testSignerContext: *signer,
		azuriteContext:    tc,
		mmrtestingConfig:  cfg,
		g:                 g,
		azuriteReader:     massifs.NewMassifReader(log, tc.Storer),
	}
}

func mustGenerateECKey(t *testing.T, curve elliptic.Curve) ecdsa.PrivateKey {
	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	require.NoError(t, err)
	return *privateKey
}

func mustNewRootSigner(t *testing.T, issuer string) massifs.RootSigner {
	cborCodec, err := massifs.NewRootSignerCodec()
	require.NoError(t, err)
	return massifs.NewRootSigner(issuer, cborCodec)
}

func signState(
	rootSigner massifs.RootSigner,
	coseSigner azkeys.IdentifiableCoseSigner, subject string, state massifs.MMRState) ([]byte, error) {

	publicKey, err := coseSigner.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("unable to get public key for signing key %w", err)
	}

	keyIdentifier := coseSigner.KeyIdentifier()
	data, err := rootSigner.Sign1(coseSigner, keyIdentifier, publicKey, subject, state, nil)
	if err != nil {
		return nil, err
	}
	return data, nil
}

type testSignerContext struct {
	key             ecdsa.PrivateKey
	rootSigner      massifs.RootSigner
	coseSigner      *azkeys.TestCoseSigner
	rootSignerCodec cbor.CBORCodec
}

func newTestSignerContext(t *testing.T, issuer string) *testSignerContext {
	var err error

	key := mustGenerateECKey(t, elliptic.P256())
	s := &testSignerContext{
		key:        key,
		rootSigner: mustNewRootSigner(t, issuer),
		coseSigner: azkeys.NewTestCoseSigner(t, key),
	}
	s.rootSignerCodec, err = massifs.NewRootSignerCodec()
	assert.NoError(t, err)

	return s
}

func (s *testSignerContext) SignedState(
	tenantIdentity string, massifIndex uint64, state massifs.MMRState,
) (*cose.CoseSign1Message, massifs.MMRState, error) {
	subject := massifs.TenantMassifBlobPath(tenantIdentity, massifIndex)
	data, err := signState(s.rootSigner, s.coseSigner, subject, state)
	if err != nil {
		return nil, massifs.MMRState{}, err
	}
	return massifs.DecodeSignedRoot(s.rootSignerCodec, data)
}

func (s *testSignerContext) SealedState(tenantIdentity string, massifIndex uint64, state massifs.MMRState) (*massifs.SealedState, error) {
	signed, state, err := s.SignedState(tenantIdentity, massifIndex, state)
	if err != nil {
		return nil, err
	}
	return &massifs.SealedState{
		Sign1Message: *signed,
		MMRState:     state,
	}, nil
}
