package massifs

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/datatrails/go-datatrails-common/logger"
)

var (
	ErrPathIsNotDir = errors.New("expected the path to be an existing directory")
)

type DirResolver interface {
	ResolveDirectory(tenantIdentityOrLocalPath string) (string, error)
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
		ctx context.Context, vc *VerifiedContext,
	) error

	InReplicaMode() bool
	GetReplicaDir() string
	GetMassifLocalPath(tenantIdentity string, massifIndex uint32) string
	GetSealLocalPath(tenantIdentity string, massifIndex uint32) string
	ResolveDirectory(tenantIdentityOrLocalPath string) (string, error)
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

func (r *LocalReader) InReplicaMode() bool {
	return r.cache.Options().replicaDir == ""
}

func (r *LocalReader) GetReplicaDir() string {
	return r.cache.Options().replicaDir
}

func (r *LocalReader) GetDirEntry(tenantIdentityOrLocalPath string) (*LogDirCacheEntry, bool) {
	return r.cache.GetEntry(tenantIdentityOrLocalPath)
}

func (r *LocalReader) ResolveDirectory(tenantIdentityOrLocalPath string) (string, error) {
	return r.cache.ResolveDirectory(tenantIdentityOrLocalPath)
}

func (r *LocalReader) ReadMassifStart(logfile string) (MassifStart, string, error) {
	return r.cache.ReadMassifStart(logfile)
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

// GetHeadVerifiedContext gets the massif and its seal and then verifies the massif
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

func (r *LocalReader) ReplaceVerifiedContext(
	ctx context.Context, vc *VerifiedContext,
) error {

	logFilename, err := r.ResolveDirectory(TenantRelativeMassifPath(vc.TenantIdentity, vc.Start.MassifIndex))
	if err != nil {
		return err
	}

	sealFilename, err := r.ResolveDirectory(TenantRelativeSealPath(vc.TenantIdentity, vc.Start.MassifIndex))
	if err != nil {
		return err
	}

	err = r.cache.ReplaceMassif(logFilename, &vc.MassifContext)
	if err != nil {
		return err
	}

	return r.cache.ReplaceSeal(sealFilename, vc.Start.MassifIndex, &SealedState{
		Sign1Message: vc.Sign1Message,
		MMRState:     vc.MMRState,
	})
}

func (r *LocalReader) GetMassif(
	ctx context.Context, tenantIdentityOrLocalPath string, massifIndex uint64,
	opts ...ReaderOption,
) (MassifContext, error) {

	// short circuit direct match, regardless of mode, to support explicit paths
	dirEntry, ok := r.cache.GetEntry(tenantIdentityOrLocalPath)
	if ok {
		return copyCachedMassifOrErr(dirEntry.ReadMassif(r.cache, massifIndex))
	}

	directory, err := r.ResolveDirectory(tenantIdentityOrLocalPath)
	if err != nil {
		return MassifContext{}, err
	}

	return copyCachedMassifOrErr(r.cache.ReadMassif(directory, massifIndex))
}

func (r *LocalReader) GetSeal(
	ctx context.Context, tenantIdentityOrLocalPath string, massifIndex uint64,
	opts ...ReaderOption,
) (SealedState, error) {

	// short circuit direct match, regardless of mode, to support explicit paths
	dirEntry, ok := r.cache.GetEntry(tenantIdentityOrLocalPath)
	if ok {
		return copyCachedSealOrErr(dirEntry.ReadSeal(r.cache, tenantIdentityOrLocalPath))
	}

	directory, err := r.ResolveDirectory(tenantIdentityOrLocalPath)
	if err != nil {
		return SealedState{}, err
	}

	return copyCachedSealOrErr(r.cache.ReadSeal(directory, massifIndex))
}

func (r *LocalReader) GetHeadMassif(
	ctx context.Context, tenantIdentityOrLocalPath string,
	opts ...ReaderOption,
) (MassifContext, error) {

	dirEntry, ok := r.cache.GetEntry(tenantIdentityOrLocalPath)
	if ok {
		return copyCachedMassifOrErr(dirEntry.ReadMassif(r.cache, uint64(dirEntry.HeadMassifIndex)))
	}

	directory, err := r.cache.ResolveDirectory(tenantIdentityOrLocalPath)
	if err != nil {
		return MassifContext{}, err
	}
	err = r.cache.FindMassifFiles(directory)
	if err != nil {
		return MassifContext{}, err
	}
	dirEntry, ok = r.cache.GetEntry(directory)
	if !ok {
		// It's a bug in FindMassifFiles if this case actually happens in practice.
		return MassifContext{}, fmt.Errorf("failed to prime directory: %s", directory)
	}
	return copyCachedMassifOrErr(dirEntry.ReadMassif(r.cache, uint64(dirEntry.HeadMassifIndex)))
}

func (r *LocalReader) GetFirstMassif(
	ctx context.Context, tenantIdentityOrLocalPath string,
	opts ...ReaderOption,
) (MassifContext, error) {
	dirEntry, ok := r.cache.GetEntry(tenantIdentityOrLocalPath)
	if ok {
		return copyCachedMassifOrErr(dirEntry.ReadMassif(r.cache, uint64(dirEntry.FirstMassifIndex)))
	}

	directory, err := r.cache.ResolveDirectory(tenantIdentityOrLocalPath)
	if err != nil {
		return MassifContext{}, err
	}
	err = r.cache.FindMassifFiles(directory)
	if err != nil {
		return MassifContext{}, err
	}
	dirEntry, ok = r.cache.GetEntry(directory)
	if !ok {
		// It's a bug in FindMassifFiles if this case actually happens in practice.
		return MassifContext{}, fmt.Errorf("failed to prime directory: %s", directory)
	}
	return copyCachedMassifOrErr(dirEntry.ReadMassif(r.cache, uint64(dirEntry.FirstMassifIndex)))
}

func (r *LocalReader) GetMassifLocalPath(tenantIdentity string, massifIndex uint32) string {
	return filepath.Join(r.GetReplicaDir(), TenantRelativeMassifPath(tenantIdentity, massifIndex))
}

func (r *LocalReader) GetSealLocalPath(tenantIdentity string, massifIndex uint32) string {
	return filepath.Join(r.GetReplicaDir(), TenantRelativeSealPath(tenantIdentity, massifIndex))
}

// GetLazyContext is an optimization for remote massif readers
// and is therefor not implemented for local massif reader
func (r *LocalReader) GetLazyContext(
	ctx context.Context, tenantIdentity string, which LogicalBlob,
	opts ...ReaderOption,
) (LogBlobContext, uint64, error) {

	return LogBlobContext{}, 0, fmt.Errorf("not implemented for local storage")
}

// copyCachedSealOrErr deals with errs from ReadSeal and returns a safe copy
// this exists to simplify the return logic flow in many methods
func copyCachedSealOrErr(cached *SealedState, err error) (SealedState, error) {
	if err != nil {
		return SealedState{}, err
	}
	return *cached, nil
}

// copyCachedMassifOrErr deals with errs from ReadMassif and returns a safe copy
// this exists to simplify the return logic flow in many methods
func copyCachedMassifOrErr(cached *MassifContext, err error) (MassifContext, error) {
	if err != nil {
		return MassifContext{}, err
	}
	return copyCachedMassif(cached), nil
}

func copyCachedMassif(cached *MassifContext) MassifContext {
	mc := *cached
	mc.peakStackMap = cached.CopyPeakStack()
	mc.Tags = cached.CopyTags()
	return mc
}
