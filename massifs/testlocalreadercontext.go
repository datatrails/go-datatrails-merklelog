//go:build integration && azurite

package massifs

import (
	"context"
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

	G mmrtesting.TestGenerator
	// We use a regular massif reader attached to azurite to test the local massif reader.
	AzuriteReader MassifReader
}

// TestLogCreatorContext holds the context data resulting from a call to CreateLog
// Unless one or more of the TEstCreateLogOptions are used, the context will not have anything interesting in it.
type TestLogCreatorContext struct {
	Preimages map[uint64][]byte
}

type TestCreateLogOption func(*TestLogCreatorContext)

func TestWithCreateLogPreImages() TestCreateLogOption {
	return func(c *TestLogCreatorContext) {
		c.Preimages = make(map[uint64][]byte)
	}
}

// CreateLog creates a log with the given tenant identity, massif height, and mmr size,
// any previous seal or massif blobs for the same tenant are first deleted
func (c *TestLocalReaderContext) CreateLog(
	tenantIdentity string, massifHeight uint8, massifCount uint32,
	opts ...TestCreateLogOption) {

	logContext := &TestLogCreatorContext{}
	for _, opt := range opts {
		opt(logContext)
	}

	generator := MMRTestingGenerateNumberedLeaf

	// If the caller needs to work with the pre-images we wrap the generator to retain them
	if logContext.Preimages != nil {
		generator = func(tenantIdentity string, base, i uint64) mmrtesting.AddLeafArgs {

			args := generator(tenantIdentity, base, i)
			logContext.Preimages[base+i] = args.Value
			return args
		}
	}

	// clear out any previous log
	c.AzuriteContext.DeleteBlobsByPrefix(TenantMassifPrefix(tenantIdentity))

	committer, err := NewTestMinimalCommitter(TestCommitterConfig{
		CommitmentEpoch: 1,
		MassifHeight:    massifHeight,
		SealOnCommit:    true, // create seals for each massif as we go
	}, c.AzuriteContext, c.G, generator)
	require.NoError(c.AzuriteContext.T, err)

	leavesPerMassif := mmr.HeightIndexLeafCount(uint64(massifHeight) - 1)

	err = committer.AddLeaves(context.TODO(), tenantIdentity, 0, leavesPerMassif*uint64(massifCount))
	require.NoError(c.AzuriteContext.T, err)
}

// AddLeavesToLog adds the requested number of leaves to the log for the given
// tenant identity.  Note the massifHeight must be the same as was provided to
// the corresponding CreateLog call
func (c *TestLocalReaderContext) AddLeavesToLog(tenantIdentity string, massifHeight uint8, leafCount int) {

	committer, err := NewTestMinimalCommitter(TestCommitterConfig{
		CommitmentEpoch: 1,
		MassifHeight:    massifHeight,
		SealOnCommit:    true, // create seals for each massif as we go
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
