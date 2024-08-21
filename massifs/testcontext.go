package massifs

import (
	"crypto/sha256"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/datatrails/go-datatrails-merklelog/mmrtesting"
)

func MMRTestingGenerateNumberedLeaf(tenantIdentity string, base, i uint64) mmrtesting.AddLeafArgs {
	h := sha256.New()
	mmrtesting.HashWriteUint64(h, base+i)

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, base+i)
	return mmrtesting.AddLeafArgs{
		Id:    0,
		AppId: b,
		Value: h.Sum(nil),
	}
}

func NewAzuriteTestContext(
	t *testing.T,
	testLabelPrefix string,
) (mmrtesting.TestContext, mmrtesting.TestGenerator, mmrtesting.TestConfig) {
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
	return tc, g, cfg
}
