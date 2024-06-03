//go:build integration && azurite

package massifs

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMassifCommitter_firstMassif covers creation of the first massive blob and related conditions
func TestMassifCommitter_firstMassif(t *testing.T) {
	var err error

	tc, g, _ := NewAzuriteTestContext(t, "Test_mmrMassifCommitter_firstMassif")
	tenantIdentity := g.NewTenantIdentity()
	clock := time.Now()
	tc.DeleteBlobsByPrefix(TenantMassifPrefix(tenantIdentity))
	fmt.Printf("delete: %d\n", time.Since(clock)/time.Millisecond)

	c := &MassifCommitter{
		Log: tc.GetLog(),
		// cfg:         tt.fields.cfg,
		Store: tc.GetStorer(),
		// lastRead:    mustNewLastReadCache(t, 1024),
	}

	var mc MassifContext
	var massifIndex uint64
	clock = time.Now()
	if mc, massifIndex, err = c.GetLastMassif(context.Background(), tenantIdentity); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	fmt.Printf("Getxx: %d\n", time.Since(clock)/time.Millisecond)
	assert.Equal(t, mc.Data == nil, true, "unexpectedly got data, probably tests re-using a container")
	assert.Equal(t, massifIndex, uint64(0))

}

func TestMassifCommitter_massifFirstContext(t *testing.T) {
	var err error

	tc, g, _ := NewAzuriteTestContext(t, "TestMassifCommitter_massifFirstContext")

	tenantIdentity := g.NewTenantIdentity()
	firstBlobPath := fmt.Sprintf("v1/mmrs/%s/0/massifs/%016d.log", tenantIdentity, 0)
	tc.DeleteBlobsByPrefix(TenantMassifPrefix(tenantIdentity))

	c := &MassifCommitter{
		Log: tc.GetLog(),
		// cfg:         tt.fields.cfg,
		Store: tc.GetStorer(),
		// lastRead:    mustNewLastReadCache(t, 1024),
	}
	var mc MassifContext
	if mc, err = c.GetCurrentContext(context.Background(), tenantIdentity, 3); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	assert.Equal(t, mc.BlobPath, firstBlobPath)
}

func TestMassifCommitter_massifAddFirst(t *testing.T) {
	var err error

	tc, g, _ := NewAzuriteTestContext(t, "TestMassifCommitter_massifAddFirst")

	tenantIdentity := g.NewTenantIdentity()
	tc.DeleteBlobsByPrefix(TenantMassifPrefix(tenantIdentity))

	MassifHeight := uint8(3)

	c := &MassifCommitter{
		Log:   tc.GetLog(),
		Store: tc.GetStorer(),
		// lastRead:    mustNewLastReadCache(t, 1024),
	}

	var mc MassifContext
	if mc, err = c.GetCurrentContext(context.Background(), tenantIdentity, MassifHeight); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	// manually insert the appropriate log entries, to seperate this test from
	// those that cover the mmr contruction and bow the massifs link together
	mc.Data = g.PadWithLeafEntries(mc.Data, 2)

	_, err = c.CommitContext(context.Background(), mc)
	assert.Nil(t, err)

	// Ensure what we read back passes the commit checks
	if mc, err = c.GetCurrentContext(context.Background(), tenantIdentity, MassifHeight); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestMassifCommitter_massifExtend(t *testing.T) {

	tc, g, _ := NewAzuriteTestContext(t, "TestMassifCommitter_massifExtend")

	var err error
	ctx := context.Background()

	tenantIdentity := g.NewTenantIdentity()
	tc.DeleteBlobsByPrefix(TenantMassifPrefix(tenantIdentity))
	MassifHeight := uint8(3)
	c := &MassifCommitter{
		Log:   tc.GetLog(),
		Store: tc.GetStorer(),
		// lastRead:    mustNewLastReadCache(t, 1024),
	}

	var mc MassifContext
	if mc, err = c.GetCurrentContext(ctx, tenantIdentity, MassifHeight); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	// add first three entries, representing the first two actual leaves and the interior root node they create
	mc.Data = g.PadWithLeafEntries(mc.Data, 3)

	_, err = c.CommitContext(ctx, mc)
	assert.Nil(t, err)

	// Ensure what we read back passes the commit checks
	if mc, err = c.GetCurrentContext(ctx, tenantIdentity, MassifHeight); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	assert.Equal(tc.T, mc.Creating, false)

	// add 3 entries, leaving space for two more logs
	mc.Data = g.PadWithLeafEntries(mc.Data, 3)
	_, err = c.CommitContext(context.Background(), mc)
	assert.Nil(t, err)

	if mc, err = c.GetCurrentContext(ctx, tenantIdentity, MassifHeight); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	assert.Equal(tc.T, mc.Creating, false)
}

func TestMassifCommitter_massifComplete(t *testing.T) {
	var err error

	tc, g, _ := NewAzuriteTestContext(t, "TestMassifCommitter_massifComplete")
	tenantIdentity := g.NewTenantIdentity()
	tc.DeleteBlobsByPrefix(TenantMassifPrefix(tenantIdentity))

	MassifHeight := uint8(3)
	c := &MassifCommitter{
		Log:   tc.GetLog(),
		Store: tc.GetStorer(),
		// lastRead:    mustNewLastReadCache(t, 1024),
	}

	var mc MassifContext
	if mc, err = c.GetCurrentContext(
		context.Background(), tenantIdentity, MassifHeight); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	// add first two entries, representing the first actual leaf and the interior root node it creates
	mc.Data = g.PadWithLeafEntries(mc.Data, 2)

	_, err = c.CommitContext(context.Background(), mc)
	assert.Nil(t, err)

	// Ensure what we read back passes the commit checks
	if mc, err = c.GetCurrentContext(context.Background(), tenantIdentity, MassifHeight); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	assert.Equal(tc.T, mc.Creating, false)

	// add 5 entries, completing the first massif
	mc.Data = g.PadWithLeafEntries(mc.Data, 5)
	_, err = c.CommitContext(context.Background(), mc)
	assert.Nil(t, err)

	if mc, err = c.GetCurrentContext(context.Background(), tenantIdentity, MassifHeight); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	assert.Equal(tc.T, mc.Creating, true)
}

// TestMassifCommitter_massifoverfilsafe tests that we can't commit a massif blob that has been over filled
func TestMassifCommitter_massifoverfilsafe(t *testing.T) {

	var err error

	tc, g, _ := NewAzuriteTestContext(t, "TestMassifCommitter_massifoverfilsafe")

	tenantIdentity := g.NewTenantIdentity()
	tc.DeleteBlobsByPrefix(TenantMassifPrefix(tenantIdentity))

	MassifHeight := uint8(3)
	c := &MassifCommitter{
		Log:   tc.GetLog(),
		Store: tc.GetStorer(),
		// lastRead:    mustNewLastReadCache(t, 1024),
	}

	var mc MassifContext
	if mc, err = c.GetCurrentContext(context.Background(), tenantIdentity, MassifHeight); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	mc.Data = g.PadWithLeafEntries(mc.Data, 2)

	_, err = c.CommitContext(context.Background(), mc)
	assert.Nil(t, err)

	// Ensure what we read back passes the commit checks
	if mc, err = c.GetCurrentContext(context.Background(), tenantIdentity, MassifHeight); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	assert.Equal(tc.T, mc.Creating, false)

	// add 3 entries, leaving space for two more logs
	mc.Data = g.PadWithLeafEntries(mc.Data, 3)
	_, err = c.CommitContext(context.Background(), mc)
	assert.Nil(t, err)

	// add 5 entries, over filling the first massif
	mc.Data = g.PadWithLeafEntries(mc.Data, 5)
	_, err = c.CommitContext(context.Background(), mc)
	if err == nil {
		tc.T.Fatalf("overfilled massif")
	}
}

func TestMassifCommitter_threemassifs(t *testing.T) {

	var err error

	ctx := context.Background()

	tc, g, _ := NewAzuriteTestContext(t, "TestMassifCommitter_threemassifs")

	tenantIdentity := g.NewTenantIdentity()
	tc.DeleteBlobsByPrefix(TenantMassifPrefix(tenantIdentity))

	MassifHeight := uint8(3)
	c := &MassifCommitter{
		Log:   tc.GetLog(),
		Store: tc.GetStorer(),
		// lastRead: mustNewLastReadCache(t, 1024),
	}

	// --- Massif 0

	var mc MassifContext
	if mc, err = c.GetCurrentContext(ctx, tenantIdentity, MassifHeight); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	// add all the entries for the first massif
	mc.Data = g.PadWithLeafEntries(mc.Data, 7)
	require.Equal(t, uint64(7), mc.RangeCount())

	_, err = c.CommitContext(context.Background(), mc)
	assert.Nil(t, err)

	// --- Massif 1

	// get the next context, it should be a 'creating' context. This is an edge
	// case as massif 0 is always exactly filled - the mmr root and the massif
	// root are the same only for this blob
	if mc, err = c.GetCurrentContext(ctx, tenantIdentity, MassifHeight); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	blobPath1 := fmt.Sprintf("v1/mmrs/%s/0/massifs/%016d.log", tenantIdentity, 1)
	assert.Equal(tc.T, mc.BlobPath, blobPath1)
	assert.Equal(tc.T, mc.Creating, true)
	assert.Equal(tc.T, len(mc.Data)-int(mc.LogStart()), 0)
	// Check our start leaf value is the last hash from the previous mmr
	assert.Equal(tc.T, mc.Start.FirstIndex, uint64(7))

	// to fill massif 1, we need to add a single alpine node (one which depends on a prior massif)
	require.Equal(t, mc.RangeCount(), uint64(7))
	mc.Data = g.PadWithLeafEntries(mc.Data, 8)
	require.Equal(t, uint64(15), mc.RangeCount())

	// commit it
	_, err = c.CommitContext(ctx, mc)
	assert.Nil(t, err)

	// --- Massif 2

	// get the context for the third, this should also be creating
	if mc, err = c.GetCurrentContext(ctx, tenantIdentity, MassifHeight); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	blobPath2 := fmt.Sprintf("v1/mmrs/%s/0/massifs/%016d.log", tenantIdentity, 2)
	assert.Equal(tc.T, mc.BlobPath, blobPath2)
	assert.Equal(tc.T, mc.Creating, true)
	assert.DeepEqual(tc.T, len(mc.Data)-int(mc.LogStart()), 0)
	assert.Equal(tc.T, mc.Start.FirstIndex, uint64(15))

	// fill it, note that this one does _not_ require an alpine node
	mc.Data = g.PadWithLeafEntries(mc.Data, 7)
	require.Equal(t, uint64(22), mc.RangeCount())

	_, err = c.CommitContext(ctx, mc)
	assert.Nil(t, err)

	// --- Massif 3
	if mc, err = c.GetCurrentContext(ctx, tenantIdentity, MassifHeight); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	assert.Equal(tc.T, mc.Start.FirstIndex, uint64(22))
	assert.Equal(tc.T, mc.Creating, true)
	blobPath3 := fmt.Sprintf("v1/mmrs/%s/0/massifs/%016d.log", tenantIdentity, 3)
	assert.Equal(tc.T, mc.BlobPath, blobPath3)

	// *part* fill it
	mc.Data = g.PadWithLeafEntries(mc.Data, 2)
	_, err = c.CommitContext(ctx, mc)
	assert.Nil(t, err)

	if mc, err = c.GetCurrentContext(ctx, tenantIdentity, MassifHeight); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	assert.Equal(tc.T, mc.Creating, false)
	assert.Equal(tc.T, mc.BlobPath, blobPath3)
	assert.Equal(tc.T, mc.Start.FirstIndex, uint64(22))
}
