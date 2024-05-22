//go:build integration && azurite

package massifs

import (
	"context"
	"crypto/sha256"
	"errors"

	"github.com/datatrails/go-datatrails-common/logger"
	"github.com/datatrails/go-datatrails-merklelog/mmrtesting"
	"github.com/stretchr/testify/require"
)

type TestCommitterConfig struct {
	CommitmentEpoch uint32
	MassifHeight    uint8
}
type TestMinimalCommitter struct {
	cfg           TestCommitterConfig
	log           logger.Logger
	g             mmrtesting.TestGenerator
	tc            mmrtesting.TestContext
	committer     MassifCommitter
	leafGenerator mmrtesting.LeafGenerator
	LastCommited  MassifContext
}

// NewTestCommitter creates a minimal forestrie leaf committer for use with
// integration testst that need to populate log content.
func NewTestMinimalCommitter(
	cfg TestCommitterConfig,
	tc mmrtesting.TestContext,
	g mmrtesting.TestGenerator,
	leafGenerator mmrtesting.LeafGenerator,
) (TestMinimalCommitter, error) {

	log := logger.Sugar.WithServiceName("merklebuilderv1")
	c := TestMinimalCommitter{
		cfg: cfg,
		log: logger.Sugar.WithServiceName("TestCommitter"),
		tc:  tc,
		g:   g,
		committer: *NewMassifCommitter(
			MassifCommitterConfig{CommitmentEpoch: cfg.CommitmentEpoch}, log, tc.GetStorer()),
		leafGenerator: leafGenerator,
	}
	return c, nil
}

func (c *TestMinimalCommitter) GetCurrentContext(
	ctx context.Context, tenantIdentity string, massifHeight uint8) (MassifContext, error) {
	return c.committer.GetCurrentContext(ctx, tenantIdentity, c.cfg.MassifHeight)
}

func (c *TestMinimalCommitter) AddLeaves(
	ctx context.Context, tenantIdentity string, base, count uint64) error {
	if count == 0 {
		return nil
	}
	mc, err := c.committer.GetCurrentContext(ctx, tenantIdentity, c.cfg.MassifHeight)
	if err != nil {
		c.log.Infof("AddLeaves: %v", err)
		return err
	}
	require.NoError(c.tc.T, err)
	batch := c.g.GenerateNumberedLeafBatch(tenantIdentity, base, count)

	hasher := sha256.New()

	for _, args := range batch {

		_, err = mc.AddHashedLeaf(hasher, args.Id, args.LogId, args.AppId, args.Value)
		if errors.Is(err, ErrMassifFull) {
			_, err = c.committer.CommitContext(ctx, mc)
			if err != nil {
				c.log.Infof("AddLeaves: %v", err)
				return err
			}
			mc, err = c.committer.GetCurrentContext(ctx, tenantIdentity, c.cfg.MassifHeight)
			if err != nil {
				c.log.Infof("AddLeaves: %v", err)
				return err
			}

			// Remember to add the leaf we failed to add above
			_, err = mc.AddHashedLeaf(
				hasher, args.Id, args.LogId, args.AppId, args.Value)
			if err != nil {
				c.log.Infof("AddLeaves: %v", err)
				return err
			}

			err = nil
		}
		if err != nil {
			c.log.Infof("AddLeaves: %v", err)
			return err
		}
	}

	c.LastCommited = mc

	_, err = c.committer.CommitContext(ctx, mc)
	if err != nil {
		c.log.Infof("AddLeaves: %v", err)
		return err
	}

	return nil
}
