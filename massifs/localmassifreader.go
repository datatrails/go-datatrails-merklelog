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
	ErrPathIsNotDir    = errors.New("expected the path to be an existing directory")
	ErrWriteIncomplete = errors.New("a file write succeeded, but the number of bytes written was shorter than the supplied data")
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
		ctx context.Context, vc *VerifiedContext, writeOpener WriteAppendOpener,
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
func (r *LocalReader) InReplicaMode() bool {
	return r.cache.Options().replicaDir != ""
}

// GetReplicaDir returns the replica directory
func (r *LocalReader) GetReplicaDir() string {
	return r.cache.Options().replicaDir
}

// GetDirEntry returns the directory entry for the given tenant identity or local path
func (r *LocalReader) GetDirEntry(tenantIdentityOrLocalPath string) (*LogDirCacheEntry, bool) {
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
	ctx context.Context, vc *VerifiedContext, writeOpener WriteAppendOpener,
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

	return r.cache.ReplaceSeal(sealFilename, vc.Start.MassifIndex, &SealedState{
		Sign1Message: vc.Sign1Message,
		MMRState:     vc.MMRState,
	})
}

// GetMassif reads the massif identified by the tenant identity and massif index
func (r *LocalReader) GetMassif(
	ctx context.Context, tenantIdentityOrLocalPath string, massifIndex uint64,
	opts ...ReaderOption,
) (MassifContext, error) {

	// short circuit direct match, regardless of mode, to support explicit paths
	dirEntry, ok := r.cache.GetEntry(tenantIdentityOrLocalPath)
	if ok {
		return copyCachedMassifOrErr(dirEntry.ReadMassif(r.cache, massifIndex))
	}

	directory, err := r.ResolveMassifDir(tenantIdentityOrLocalPath)
	if err != nil {
		return MassifContext{}, err
	}

	return copyCachedMassifOrErr(r.cache.ReadMassif(directory, massifIndex))
}

// GetSeal reads the seal identified by the tenant identity and massif index
func (r *LocalReader) GetSeal(
	ctx context.Context, tenantIdentityOrLocalPath string, massifIndex uint64,
	opts ...ReaderOption,
) (SealedState, error) {

	// short circuit direct match, regardless of mode, to support explicit paths
	dirEntry, ok := r.cache.GetEntry(tenantIdentityOrLocalPath)
	if ok {
		return copyCachedSealOrErr(dirEntry.ReadSeal(r.cache, tenantIdentityOrLocalPath))
	}

	directory, err := r.ResolveSealDir(tenantIdentityOrLocalPath)
	if err != nil {
		return SealedState{}, err
	}

	return copyCachedSealOrErr(r.cache.ReadSeal(directory, massifIndex))
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

	dirEntry, ok := r.cache.GetEntry(tenantIdentityOrLocalPath)
	if ok {
		return copyCachedMassifOrErr(dirEntry.ReadMassif(r.cache, uint64(dirEntry.HeadMassifIndex)))
	}

	directory, err := r.cache.ResolveMassifDir(tenantIdentityOrLocalPath)
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

	directory, err := r.cache.ResolveMassifDir(tenantIdentityOrLocalPath)
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
