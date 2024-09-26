//go:build integration && azurite

package massifs

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/datatrails/go-datatrails-common/azblob"
	"github.com/datatrails/go-datatrails-common/azkeys"
	"github.com/datatrails/go-datatrails-common/logger"
	"github.com/datatrails/go-datatrails-merklelog/mmr"
	"github.com/datatrails/go-datatrails-merklelog/mmrtesting"
	"github.com/stretchr/testify/require"
)

type TestCommitterConfig struct {
	CommitmentEpoch uint32
	MassifHeight    uint8
	SealOnCommit    bool
}
type TestMinimalCommitter struct {
	cfg           TestCommitterConfig
	log           logger.Logger
	g             mmrtesting.TestGenerator
	tc            mmrtesting.TestContext
	committer     MassifCommitter
	leafGenerator mmrtesting.LeafGenerator
	LastCommited  MassifContext

	SealIssuer   string
	RootSigner   RootSigner
	CoseSigner   *azkeys.TestCoseSigner
	SealerKey    *ecdsa.PrivateKey
	SealerPubKey *ecdsa.PublicKey
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
	if !c.cfg.SealOnCommit {
		return c, nil
	}

	c.SealIssuer = "seal.datatrails.ai"
	key := TestGenerateECKey(tc.T, elliptic.P256())
	c.SealerKey = &key
	c.CoseSigner = azkeys.NewTestCoseSigner(tc.T, key)
	codec, err := NewRootSignerCodec()
	require.NoError(tc.T, err)
	c.RootSigner = NewRootSigner(c.SealIssuer, codec)
	return c, nil
}

func (c *TestMinimalCommitter) GetCurrentContext(
	ctx context.Context, tenantIdentity string, massifHeight uint8) (MassifContext, error) {
	return c.committer.GetCurrentContext(ctx, tenantIdentity, c.cfg.MassifHeight)
}

// ContextCommitted seals the current massif context if the context is configure with SealOnCommit
func (c *TestMinimalCommitter) ContextCommitted(ctx context.Context, tenantIdentity string, mc MassifContext) error {
	if !c.cfg.SealOnCommit {
		return nil
	}

	mmrSize := mc.RangeCount()
	if mmrSize == 0 {
		return errors.New("no leaves to seal")
	}
	peaks, err := mmr.PeakHashes(&mc, mmrSize)
	if err != nil {
		return err
	}

	state := MMRState{
		MMRSize:         mmrSize,
		Peaks:           peaks,
		Timestamp:       time.Now().UnixMilli(),
		CommitmentEpoch: c.cfg.CommitmentEpoch,
		IDTimestamp:     mc.GetLastIdTimestamp(),
	}

	subject := TenantMassifBlobPath(tenantIdentity, uint64(mc.Start.MassifIndex))
	publicKey, err := c.CoseSigner.PublicKey()
	if err != nil {
		return fmt.Errorf("unable to get public key for signing key %w", err)
	}

	keyIdentifier := c.CoseSigner.KeyIdentifier()
	data, err := c.RootSigner.Sign1(c.CoseSigner, keyIdentifier, publicKey, subject, state, nil)
	if err != nil {
		return err
	}

	blobPath := TenantMassifSignedRootPath(tenantIdentity, mc.Start.MassifIndex)

	// Ensure we set the tag for the last id timestamp covered by the seal. This
	// supports efficient discovery of "logs to be sealed" both internally and
	// by independent verifiers.

	lastid := IDTimestampToHex(state.IDTimestamp, uint8(mc.Start.CommitmentEpoch))
	tags := map[string]string{}
	tags[TagKeyLastID] = lastid

	// just put it hard, without the etag check
	_, err = c.committer.Store.Put(ctx, blobPath, azblob.NewBytesReaderCloser(data), azblob.WithTags(tags))
	if err != nil {
		return err
	}
	return nil
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
	err = mc.CreatePeakStackMap()
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
			err = c.ContextCommitted(ctx, tenantIdentity, mc)
			if err != nil {
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
	err = c.ContextCommitted(ctx, tenantIdentity, mc)
	if err != nil {
		return err
	}

	return nil
}
