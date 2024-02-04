package testing

import (
	"context"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/datatrails/go-datatrails-common/azblob"
	"github.com/datatrails/go-datatrails-common/logger"
	"github.com/stretchr/testify/require"
)

type TestContext struct {
	TestGenerator
	Log    logger.Logger
	Storer *azblob.Storer
	T      *testing.T
}

// XXX: TODO TenantMassifPrefix duplicated here to avoid import cycle. refactor
const (
	ValueBytes       = 32
	V1MMRPrefix      = "v1/mmrs"
	V1MMRBlobNameFmt = "%016d.log"
)

func TenantMassifPrefix(tenantIdentity string) string {
	return fmt.Sprintf(
		"%s/%s/massifs/", V1MMRPrefix, tenantIdentity,
	)

}

type TestConfig struct {
	// We seed the RNG of the provided StartTimeMS. It is normal to force it to
	// some fixed value so that the generated data is the same from run to run.
	StartTimeMS     int64
	EventRate       int
	TestLabelPrefix string
	TenantIdentity  string // can be ""
	Container       string // can be "" defaults to TestLablePrefix
}

func NewContext(t *testing.T, cfg TestConfig) TestContext {
	c := TestContext{
		TestGenerator: NewTestGenerator(
			t, cfg.StartTimeMS/1000, TestGeneratorConfig{
				StartTimeMS:     cfg.StartTimeMS,
				EventRate:       cfg.EventRate,
				TenantIdentity:  cfg.TenantIdentity,
				TestLabelPrefix: cfg.TestLabelPrefix,
			},
		),
		T: t,
	}
	logger.New("TEST")
	c.Log = logger.Sugar.WithServiceName(cfg.TestLabelPrefix)

	container := cfg.Container
	if container == "" {
		container = cfg.TestLabelPrefix
	}

	var err error
	c.Storer, err = azblob.NewDev(azblob.NewDevConfigFromEnv(), container)
	if err != nil {
		t.Fatalf("failed to connect to blob store emulator: %v", err)
	}
	client := c.Storer.GetServiceClient()
	// Note: we expect a 'already exists' error here and  ignore it.
	_, _ = client.CreateContainer(context.Background(), container, nil)

	return c
}

func (c *TestContext) GetLog() logger.Logger { return c.Log }

func (c *TestContext) GetStorer() *azblob.Storer {
	return c.Storer
}

func (c *TestContext) DeleteTenantMassifs(tenantIdentity string) {

	var err error
	var r *azblob.ListerResponse
	var blobs []string

	blobPrefixPath := TenantMassifPrefix(tenantIdentity)
	var marker azblob.ListMarker
	for {
		r, err = c.Storer.List(
			context.Background(),
			azblob.WithListPrefix(blobPrefixPath), azblob.WithListMarker(marker) /*, azblob.WithListTags()*/)

		require.NoError(c.T, err)

		for _, i := range r.Items {
			blobs = append(blobs, *i.Name)
		}
		if len(r.Items) == 0 || r.Marker == nil {
			break
		}
		marker = r.Marker
	}
	for _, blobPath := range blobs {
		err = c.Storer.Delete(context.Background(), blobPath)
		require.NoError(c.T, err)
	}
}

func (c *TestContext) PadWithNumberedLeaves(data []byte, first, n int) []byte {
	if n == 0 {
		return data
	}
	values := make([]byte, ValueBytes*n)
	for i := 0; i < n; i++ {
		binary.BigEndian.PutUint32(values[i*ValueBytes+ValueBytes-4:i*ValueBytes+ValueBytes], uint32(first+i))
	}
	return append(data, values...)
}
