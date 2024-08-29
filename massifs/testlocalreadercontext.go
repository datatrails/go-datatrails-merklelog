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

// CreateLog creates a log with the given tenant identity, massif height, and mmr size,
// any previous seal or massif blobs for the same tenant are first deleted
func (c *TestLocalReaderContext) CreateLog(tenantIdentity string, massifHeight uint8, massifCount uint32) {

	// clear out any previous log
	c.AzuriteContext.DeleteBlobsByPrefix(TenantMassifPrefix(tenantIdentity))

	committer, err := NewTestMinimalCommitter(TestCommitterConfig{
		CommitmentEpoch: 1,
		MassifHeight:    massifHeight,
		SealOnCommit:    true, // create seals for each massif as we go
	}, c.AzuriteContext, c.G, MMRTestingGenerateNumberedLeaf)
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
