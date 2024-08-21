package massifs

import (
	"context"
	"fmt"
	"strings"
	"testing"

	azStorageBlob "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/datatrails/go-datatrails-common/azblob"
	"github.com/datatrails/go-datatrails-common/logger"
	"github.com/datatrails/go-datatrails-merklelog/mmrtesting"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tenantMMRPrefix = "v1/mmrs/tenant/"
)

func fmtTenantBlobPath(tenantUUID string, blobName string) string {
	return fmt.Sprintf("%s%s/%s", tenantMMRPrefix, tenantUUID, blobName)
}

func parseTenantUUID(path string) (string, error) {
	tenantPath := strings.Replace(path, tenantMMRPrefix, "", 1)
	before, _, ok := strings.Cut(tenantPath, "/")
	if !ok {
		return "", fmt.Errorf("separator not found in %s", path)
	}
	_, err := uuid.Parse(before)
	if err != nil {
		return "", err
	}
	return before, nil
}

func TestEnumerateTenants(t *testing.T) {
	logger.New("NOOP")
	defer logger.OnExit()

	seed := int64((1698342521) * 1000)
	g := mmrtesting.NewTestGenerator(t, seed, mmrtesting.TestGeneratorConfig{
		StartTimeMS: seed, EventRate: 500,
		TestLabelPrefix: "TestEnumerateTenants"}, MMRTestingGenerateNumberedLeaf)

	type args struct {
		batches []tenantBatch
	}
	tests := []struct {
		name      string
		args      args
		wantCount int
		wantErr   bool
	}{
		{name: "one batch, one tenant", args: args{
			batches: []tenantBatch{{itemCount: 4}}}},
		{name: "one batch, two tenants", args: args{
			batches: []tenantBatch{{mixCount: 2, itemCount: 4}}}},
		{name: "two batches, three tenants", args: args{
			batches: []tenantBatch{{itemCount: 4}, {mixCount: 2, itemCount: 4}}}},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var newUUIDs []string
			var marker azblob.ListMarker
			var err error

			store := newReaderForBatches(&g, tt.args.batches)
			found := map[string]any{}

			expectVisited := 0

			for iBatch := 0; iBatch < len(tt.args.batches); iBatch++ {

				newUUIDs, marker, err = EnumerateIdentifiedPaths(
					ctx, store, "ignored", parseTenantUUID, found, marker)
				assert.NoError(t, err)
				store.NextBatch()
				expectCount := tt.args.batches[iBatch].mixCount
				if expectCount == 0 {
					expectCount = 1
				}
				assert.Equal(t, len(newUUIDs), expectCount)
				expectVisited += expectCount
			}
			assert.Equal(t, len(found), expectVisited)
		})
	}
}

type tenantBatch struct {
	mixCount  int
	itemCount int
}

func newReaderForBatches(
	g *mmrtesting.TestGenerator, batches []tenantBatch) testEnumTenantsReader {
	r := testEnumTenantsReader{}
	totalTenantCount := 0
	for _, batch := range batches {
		if batch.mixCount == 0 {
			batch.mixCount = 1
		}
		r.batches = append(
			r.batches, batchMixedTenant(
				g, totalTenantCount, batch.itemCount, batch.mixCount))
		totalTenantCount += batch.mixCount
	}
	return r
}

// batchMixedTenants creates a batched list blob response with blob items for
// the specified number of tenants present
func batchMixedTenant(g *mmrtesting.TestGenerator, base, responseItemCount, mixCount int) *azblob.ListerResponse {

	require.Less(g.T, mixCount, responseItemCount)
	stripeSize := responseItemCount/mixCount + 1

	tenantID := g.NewRandomUUIDString(g.T)
	var items []*azStorageBlob.BlobItemInternal
	for i := 0; i < responseItemCount; i++ {
		name := fmtTenantBlobPath(tenantID, fmt.Sprintf("%d", base+i))
		items = append(items, &azStorageBlob.BlobItemInternal{
			Name: &name,
		})
		if (i+1)%stripeSize == 0 {
			tenantID = g.NewRandomUUIDString(g.T)
		}
	}
	return &azblob.ListerResponse{
		Items: items,
		// Caller can setup Marker as required.
	}
}

// batchSameTenant creates a batched list blob response where all items share the same tenant
func batchSameTenant(g *mmrtesting.TestGenerator, base, responseItemCount int) *azblob.ListerResponse {

	tenantID := g.NewRandomUUIDString(g.T)
	var items []*azStorageBlob.BlobItemInternal
	for i := 0; i < responseItemCount; i++ {
		name := fmtTenantBlobPath(tenantID, fmt.Sprintf("%d", base+i))
		items = append(items, &azStorageBlob.BlobItemInternal{
			Name: &name,
		})
	}
	return &azblob.ListerResponse{
		Items: items,
		// Caller can setup Marker as required.
	}
}

type testEnumTenantsReader struct {
	batches   []*azblob.ListerResponse
	nextBatch int
}

func (r *testEnumTenantsReader) NextBatch() {
	r.nextBatch += 1
}

func (r testEnumTenantsReader) Reader(
	ctx context.Context,
	identity string,
	opts ...azblob.Option,
) (*azblob.ReaderResponse, error) {
	return nil, fmt.Errorf("this unit test suite only needs the List implementation")
}

func (r testEnumTenantsReader) List(
	ctx context.Context, opts ...azblob.Option) (*azblob.ListerResponse, error) {

	if r.nextBatch >= len(r.batches) {
		return nil, fmt.Errorf("ran out of test batches")
	}
	// note: because the List implementation interface has a by value receiver,
	// we can't increment nextBatch here
	batch := r.batches[r.nextBatch]
	return batch, nil
}

func (r testEnumTenantsReader) FilteredList(
	ctx context.Context, tagsFilter string, opts ...azblob.Option) (*azblob.FilterResponse, error) {

	if r.nextBatch >= len(r.batches) {
		return nil, fmt.Errorf("ran out of test batches")
	}
	// note: because the List implementation interface has a by value receiver,
	// we can't increment nextBatch here
	// batch := r.batches[r.nextBatch]
	items := make([]*azStorageBlob.FilterBlobItem, 0, len(r.batches[r.nextBatch].Items))
	for _, it := range r.batches[r.nextBatch].Items {
		filterItem := &azStorageBlob.FilterBlobItem{
			// ignore container name for now
			Name: it.Name,
			Tags: it.BlobTags,
		}
		items = append(items, filterItem)
	}
	batch := &azblob.FilterResponse{
		Items: items,
		// Caller can setup Marker as required.
	}

	return batch, nil
}
