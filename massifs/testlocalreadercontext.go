//go:build integration && azurite

package massifs

import (
	"context"
	"crypto/ecdsa"
	"strings"
	"testing"

	"github.com/datatrails/go-datatrails-common/logger"
	"github.com/datatrails/go-datatrails-merklelog/mmr"
	"github.com/datatrails/go-datatrails-merklelog/mmrtesting"
	"github.com/stretchr/testify/require"
)

type TestLocalReaderContext struct {
	TestSignerContext
	AzuriteContext  mmrtesting.TestContext
	TestConfig      mmrtesting.TestConfig
	CommitterConfig TestCommitterConfig
	LogOptions      TestLogOptions

	G mmrtesting.TestGenerator
	// We use a regular massif reader attached to azurite to test the local massif reader.
	AzuriteReader MassifReader
}

// TestLogOptions holds the context data resulting from a call to CreateLog
// Unless one or more of the TEstCreateLogOptions are used, the context will not have anything interesting in it.
type TestLogOptions struct {
	Preimages        map[uint64][]byte
	ContinueExisting bool // if true, don'
	SealKey          *ecdsa.PrivateKey
	UsingV0Seals     bool
}

type TestLogOption func(*TestLogOptions)

func TestWithCreateLogPreImages() TestLogOption {
	return func(c *TestLogOptions) {
		c.Preimages = make(map[uint64][]byte)
	}
}

func TestWithSealKey(key *ecdsa.PrivateKey) TestLogOption {
	return func(c *TestLogOptions) {
		c.SealKey = key
	}
}

func TestWithV0Seals() TestLogOption {
	return func(c *TestLogOptions) {
		c.UsingV0Seals = true
	}
}

// CreateLog creates a log with the given tenant identity, massif height, and mmr size,
// any previous seal or massif blobs for the same tenant are first deleted
// To create a single incomplete massif, call AddLeavesToLog with a leafCount instead of CreateLog
func (c *TestLocalReaderContext) CreateLog(
	tenantIdentity string, massifHeight uint8, massifCount uint32,
	opts ...TestLogOption) {

	options := &TestLogOptions{}
	for _, opt := range opts {
		opt(options)
	}

	generator := MMRTestingGenerateNumberedLeaf

	// If the caller needs to work with the pre-images we wrap the generator to retain them
	if options.Preimages != nil {
		generator = func(tenantIdentity string, base, i uint64) mmrtesting.AddLeafArgs {

			args := generator(tenantIdentity, base, i)
			options.Preimages[base+i] = args.Value
			return args
		}
	}

	// clear out any previous log
	c.AzuriteContext.DeleteBlobsByPrefix(TenantMassifPrefix(tenantIdentity))

	committer, err := NewTestMinimalCommitter(TestCommitterConfig{
		CommitmentEpoch: 1,
		MassifHeight:    massifHeight,
		SealOnCommit:    true, // create seals for each massif as we go
		SealerKey:       options.SealKey,
		UseV0Seals:      options.UsingV0Seals,
	}, c.AzuriteContext, c.G, generator)
	require.NoError(c.AzuriteContext.T, err)

	leavesPerMassif := mmr.HeightIndexLeafCount(uint64(massifHeight) - 1)

	// using base = 0 means the caller can't predict the leaf hash based on the mmr index, but otherwise it's fine
	// if the caller is overriding the generator, they can do what they like
	err = committer.AddLeaves(context.TODO(), tenantIdentity, 0, leavesPerMassif*uint64(massifCount))
	require.NoError(c.AzuriteContext.T, err)
}

// AddLeavesToLog adds the requested number of leaves to the log for the given
// tenant identity.  Note the massifHeight must be the same as was provided to
// the corresponding CreateLog call
// To create a single incomplete massif, call AddLeavesToLog with a leafCount instead of CreateLog
func (c *TestLocalReaderContext) AddLeavesToLog(tenantIdentity string, massifHeight uint8, leafCount int, opts ...TestLogOption) {

	options := &TestLogOptions{}
	for _, opt := range opts {
		opt(options)
	}
	generator := MMRTestingGenerateNumberedLeaf

	// If the caller needs to work with the pre-images we wrap the generator to retain them
	if options.Preimages != nil {
		generator = func(tenantIdentity string, base, i uint64) mmrtesting.AddLeafArgs {

			args := generator(tenantIdentity, base, i)
			options.Preimages[base+i] = args.Value
			return args
		}
	}

	committer, err := NewTestMinimalCommitter(TestCommitterConfig{
		CommitmentEpoch: 1,
		MassifHeight:    massifHeight,
		SealOnCommit:    true, // create seals for each massif as we go
		SealerKey:       options.SealKey,
		UseV0Seals:      options.UsingV0Seals,
	}, c.AzuriteContext, c.G, MMRTestingGenerateNumberedLeaf)
	require.NoError(c.AzuriteContext.T, err)

	err = committer.AddLeaves(context.TODO(), tenantIdentity, 0, uint64(leafCount))
	require.NoError(c.AzuriteContext.T, err)
}

func NewLocalMassifReaderTestContext(
	t *testing.T, log logger.Logger, testLabelPrefix string) TestLocalReaderContext {
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
		MMRTestingGenerateNumberedLeaf,
	)

	signer := NewTestSignerContext(t, testLabelPrefix)
	return TestLocalReaderContext{
		TestSignerContext: *signer,
		AzuriteContext:    tc,
		TestConfig:        cfg,
		G:                 g,
		AzuriteReader:     NewMassifReader(log, tc.Storer),
	}
}
