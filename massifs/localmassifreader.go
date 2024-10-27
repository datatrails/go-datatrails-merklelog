package massifs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/datatrails/go-datatrails-common/cose"
	"github.com/datatrails/go-datatrails-common/logger"
)

var (
	ErrPathIsNotDir             = errors.New("expected the path to be an existing directory")
	ErrWriteIncomplete          = errors.New("a file write succeeded, but the number of bytes written was shorter than the supplied data")
	ErrFailedToCreateReplicaDir = errors.New("failed to create a directory needed for local replication")
)

type DirResolver interface {
	ResolveMassifDir(tenantIdentityOrLocalPath string) (string, error)
	ResolveSealDir(tenantIdentityOrLocalPath string) (string, error)
}
type WriteAppendOpener interface {
	Open(string) (io.WriteCloser, error)
}

type VerifiedContextReader interface {
	GetVerifiedContext(
		ctx context.Context, tenantIdentity string, massifIndex uint64,
		opts ...ReaderOption,
	) (*VerifiedContext, error)
}

type ReplicaReader interface {
	VerifiedContextReader
	GetHeadVerifiedContext(
		ctx context.Context, tenantIdentity string,
		opts ...ReaderOption,
	) (*VerifiedContext, error)
	ReplaceVerifiedContext(
		vc *VerifiedContext, writeOpener WriteAppendOpener,
	) error

	InReplicaMode() bool
	GetReplicaDir() string
	EnsureReplicaDirs(tenantIdentity string) error
	GetMassifLocalPath(tenantIdentity string, massifIndex uint32) string
	GetSealLocalPath(tenantIdentity string, massifIndex uint32) string
	ResolveMassifDir(tenantIdentityOrLocalPath string) (string, error)
	ResolveSealDir(tenantIdentityOrLocalPath string) (string, error)
}

type LocalReader struct {
	log logger.Logger
	// cache of previously read material, this is typically shared with a LocalSealReader instance
	cache DirCache
}

func NewLocalReader(
	log logger.Logger, cache DirCache,
) (LocalReader, error) {
	r := LocalReader{
		log:   log,
		cache: cache,
	}
	return r, nil
}

// InReplicaMode returns true if the reader is in replica mode
//
// Replica mode shoud be used when efficient, consistent, replication of logs
// for multiple tenants is desired. In replica mode, the repilcated logs are
// maintained in a tree under the configured ReplicaDir, using a path scheme
// that matches datatrails remote schema.
func (r *LocalReader) InReplicaMode() bool {
	return r.cache.Options().replicaDir != ""
}

// GetReplicaDir returns the replica directory
// This is significant when maitaining replicas for multiple tenants. See InReplicaMode for more information
func (r *LocalReader) GetReplicaDir() string {
	return r.cache.Options().replicaDir
}

// GetDirEntry returns the directory entry for the given tenant identity or local path
func (r *LocalReader) GetDirEntry(tenantIdentityOrLocalPath string) (DirCacheEntry, bool) {
	return r.cache.GetEntry(tenantIdentityOrLocalPath)
}

// ResolveMassifDir resolves the tenant identity or local path to a directory
func (r *LocalReader) ResolveMassifDir(tenantIdentityOrLocalPath string) (string, error) {
	return r.cache.ResolveMassifDir(tenantIdentityOrLocalPath)
}

// ResolveSealDir resolves the tenant identity or local path to a directory
func (r *LocalReader) ResolveSealDir(tenantIdentityOrLocalPath string) (string, error) {
	return r.cache.ResolveSealDir(tenantIdentityOrLocalPath)
}

// ReadMassifStart reads the massif start from the given log file
func (r *LocalReader) ReadMassifStart(logfile string) (MassifStart, string, error) {
	return r.cache.ReadMassifStart(logfile)
}

// EnsureReplicaDirs ensures the replica directories exist for the given tenant identity
func (r *LocalReader) EnsureReplicaDirs(tenantIdentity string) error {
	if !r.InReplicaMode() {
		return fmt.Errorf("replica dir must be configured on the local reader")
	}

	massifsDir := filepath.Dir(r.GetMassifLocalPath(tenantIdentity, 0))
	sealsDir := filepath.Dir(r.GetSealLocalPath(tenantIdentity, 0))

	err := os.MkdirAll(massifsDir, os.FileMode(0755))
	if err != nil {
		return fmt.Errorf("%w: %s", ErrFailedToCreateReplicaDir, massifsDir)

	}
	err = os.MkdirAll(sealsDir, os.FileMode(0755))
	if err != nil {
		return fmt.Errorf("%w: %s", ErrFailedToCreateReplicaDir, sealsDir)

	}
	return nil
}

// GetVerifiedContext gets the massif and its seal and then verifies the massif
// data against the seal. If the caller provides the expected public key, the
// public key on the seal is required to match
func (r *LocalReader) GetVerifiedContext(
	ctx context.Context, tenantIdentity string, massifIndex uint64,
	opts ...ReaderOption,
) (*VerifiedContext, error) {

	options, err := checkedVerifiedContextOptions(r.cache.Options().ReaderOptions, opts...)
	if err != nil {
		return nil, err
	}

	mc, err := r.GetMassif(ctx, tenantIdentity, massifIndex, opts...)
	if err != nil {
		return nil, err
	}

	return mc.verifyContext(ctx, options)
}

// GetHeadVerifiedContext gets the head massif and its seal and then verifies the massif
// data against the seal. If the caller provides the expected public key, the
// public key on the seal is required to match
func (r *LocalReader) GetHeadVerifiedContext(
	ctx context.Context, tenantIdentity string,
	opts ...ReaderOption,
) (*VerifiedContext, error) {

	options, err := checkedVerifiedContextOptions(r.cache.Options().ReaderOptions, opts...)
	if err != nil {
		return nil, err
	}

	mc, err := r.GetHeadMassif(ctx, tenantIdentity, opts...)
	if err != nil {
		return nil, err
	}

	return mc.verifyContext(ctx, options)
}

// VerifyContext verifies an arbitrary context and returns a verified context if this succeeds.
func (r *LocalReader) VerifyContext(
	ctx context.Context, mc MassifContext,
	opts ...ReaderOption,
) (*VerifiedContext, error) {

	options, err := checkedVerifiedContextOptions(r.cache.Options().ReaderOptions, opts...)
	if err != nil {
		return nil, err
	}
	return mc.verifyContext(ctx, options)
}

// ReplaceVerifiedContext writes the content from the verified remote to the
// local replica.  is the callers responsibility to ensure the context was
// verified, and that the writeOpener opens the file in append mode if it
// already exists.
func (r *LocalReader) ReplaceVerifiedContext(
	vc *VerifiedContext, writeOpener WriteAppendOpener,
) error {

	logFilename := r.GetMassifLocalPath(vc.TenantIdentity, vc.Start.MassifIndex)
	err := writeAll(writeOpener, logFilename, vc.Data)
	if err != nil {
		return err
	}

	sealFilename := r.GetSealLocalPath(vc.TenantIdentity, vc.Start.MassifIndex)
	if err != nil {
		return err
	}
	sealBytes, err := vc.Sign1Message.MarshalCBOR()
	if err != nil {
		return err
	}
	err = writeAll(writeOpener, sealFilename, sealBytes)
	if err != nil {
		return err
	}

	err = r.cache.ReplaceMassif(logFilename, &vc.MassifContext)
	if err != nil {
		return err
	}

	// Note: ensure that the root is *never* written to disc or available in the
	// cached copy of the seal, so that it always has to be recomputed.
	state := vc.MMRState
	state.LegacySealRoot = nil
	state.Peaks = nil

	return r.cache.ReplaceSeal(sealFilename, vc.Start.MassifIndex, &SealedState{
		Sign1Message: vc.Sign1Message,
		MMRState:     state,
	})
}

// reconcileCachedContextTenantIdentity ensures that the tenant identity is set on the context
// Of, if already set, that it matches the callers expectation. This is
// necessary to deal with the fact that the tenant identity cant be inferred
// from the storage path in all cases.
func reconcileCachedContextTenantIdentity(mc *MassifContext, tenantIdentityOrLocalPath string) error {
	if mc.TenantIdentity == "" {
		mc.TenantIdentity = tenantIdentityOrLocalPath
		return nil
	}
	if mc.TenantIdentity != tenantIdentityOrLocalPath {
		return ErrTenantIdInconsistent
	}
	return nil
}

// GetMassif reads the massif identified by the tenant identity and massif index
func (r *LocalReader) GetMassif(
	ctx context.Context, tenantIdentityOrLocalPath string, massifIndex uint64,
	opts ...ReaderOption,
) (MassifContext, error) {

	var mc *MassifContext

	dirEntry, err := r.resolveMassifDirEntry(tenantIdentityOrLocalPath)
	if err != nil {
		return MassifContext{}, err
	}

	if mc, err = dirEntry.ReadMassif(r.cache, massifIndex); err != nil {
		return MassifContext{}, err
	}

	// support situations where the context tenant can't be infered from the storage path
	if err = reconcileCachedContextTenantIdentity(mc, tenantIdentityOrLocalPath); err != nil {
		return MassifContext{}, err
	}

	return copyCachedMassif(mc), nil
}

// GetSeal reads the seal identified by the tenant identity and massif index
func (r *LocalReader) GetSeal(
	ctx context.Context, tenantIdentityOrLocalPath string, massifIndex uint64,
	opts ...ReaderOption,
) (SealedState, error) {

	dirEntry, err := r.resolveSealDirEntry(tenantIdentityOrLocalPath)
	if err != nil {
		return SealedState{}, err
	}

	sstate, err := dirEntry.GetSeal(r.cache, massifIndex)
	if err != nil {
		return SealedState{}, err
	}
	return *sstate, nil
}

// GetSignedRoot satisfies the SealGetter interface.
// This is the default seal getter for the local reader when GetVerifiedContext is called.
func (r *LocalReader) GetSignedRoot(
	ctx context.Context, tenantIdentityOrLocalPath string, massifIndex uint32,
	opts ...ReaderOption,
) (*cose.CoseSign1Message, MMRState, error) {
	sealedState, err := r.GetSeal(ctx, tenantIdentityOrLocalPath, uint64(massifIndex), opts...)
	if err != nil {
		return nil, MMRState{}, err
	}
	return &sealedState.Sign1Message, sealedState.MMRState, nil
}

// GetHeadMassif reads the most recent massif in the log identified by the tenant identity
func (r *LocalReader) GetHeadMassif(
	ctx context.Context, tenantIdentityOrLocalPath string,
	opts ...ReaderOption,
) (MassifContext, error) {

	var mc *MassifContext

	dirEntry, err := r.resolveMassifDirEntry(tenantIdentityOrLocalPath)
	if err != nil {
		return MassifContext{}, err
	}

	if mc, err = dirEntry.ReadMassif(r.cache, uint64(dirEntry.GetInfo().HeadMassifIndex)); err != nil {
		return MassifContext{}, err
	}

	// support situations where the context tenant can't be infered from the storage path
	if err = reconcileCachedContextTenantIdentity(mc, tenantIdentityOrLocalPath); err != nil {
		return MassifContext{}, err
	}

	return copyCachedMassif(mc), nil
}

func (r *LocalReader) GetFirstMassif(
	ctx context.Context, tenantIdentityOrLocalPath string,
	opts ...ReaderOption,
) (MassifContext, error) {

	var mc *MassifContext

	dirEntry, err := r.resolveMassifDirEntry(tenantIdentityOrLocalPath)
	if err != nil {
		return MassifContext{}, err
	}
	if mc, err = dirEntry.ReadMassif(r.cache, uint64(dirEntry.GetInfo().FirstMassifIndex)); err != nil {
		return MassifContext{}, err
	}
	// support situations where the context tenant can't be infered from the storage path
	if err = reconcileCachedContextTenantIdentity(mc, tenantIdentityOrLocalPath); err != nil {
		return MassifContext{}, err
	}

	return copyCachedMassif(mc), nil
}

func (r *LocalReader) resolveMassifDirEntry(tenantIdentityOrLocalPath string) (DirCacheEntry, error) {

	directory, err := r.cache.ResolveMassifDir(tenantIdentityOrLocalPath)
	if err != nil {
		return nil, err
	}

	dirEntry, err := r.cache.ReadMassifDirEntry(directory)
	if err != nil {
		return nil, err
	}
	// The DirCacheEntry exists to facilitate mocked testing, this method has to return a concrete type
	return dirEntry, nil
}

func (r *LocalReader) resolveSealDirEntry(tenantIdentityOrLocalPath string) (*LogDirCacheEntry, error) {

	directory, err := r.cache.ResolveSealDir(tenantIdentityOrLocalPath)
	if err != nil {
		return nil, err
	}

	dirEntry, err := r.cache.ReadSealDirEntry(directory)
	if err != nil {
		return nil, err
	}
	// The DirCacheEntry exists to facilitate mocked testing, this method has to return a concrete type
	return dirEntry.(*LogDirCacheEntry), nil
}

// GetMassifLocalPath returns the local path for the massif identified by the
// tenant identity and massif index
func (r *LocalReader) GetMassifLocalPath(tenantIdentity string, massifIndex uint32) string {
	return filepath.Join(r.GetReplicaDir(), ReplicaRelativeMassifPath(tenantIdentity, massifIndex))
}

// GetSealLocalPath returns the local path for the seal identified by the tenant identity and massif index
func (r *LocalReader) GetSealLocalPath(tenantIdentity string, massifIndex uint32) string {
	return filepath.Join(r.GetReplicaDir(), ReplicaRelativeSealPath(tenantIdentity, massifIndex))
}

// GetLazyContext is an optimization for remote massif readers
// and is therefor not implemented for local massif reader
func (r *LocalReader) GetLazyContext(
	ctx context.Context, tenantIdentity string, which LogicalBlob,
	opts ...ReaderOption,
) (LogBlobContext, uint64, error) {

	return LogBlobContext{}, 0, fmt.Errorf("not implemented for local storage")
}

func copyCachedMassif(cached *MassifContext) MassifContext {
	mc := *cached
	mc.peakStackMap = cached.CopyPeakStack()
	mc.Tags = cached.CopyTags()
	return mc
}

func writeAll(wo WriteAppendOpener, filename string, data []byte) error {
	f, err := wo.Open(filename)
	if err != nil {
		return err

	}
	defer f.Close()

	n, err := f.Write(data)
	if err != nil {
		return err
	}

	if n != len(data) {
		return fmt.Errorf("%w: %s", ErrWriteIncomplete, filename)
	}
	return nil
}
