package massifs

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/datatrails/go-datatrails-common/cbor"
	"github.com/datatrails/go-datatrails-common/logger"
)

var (
	ErrPathIsNotDir = errors.New("expected the path to be an existing directory")
)

type DirResolver interface {
	ResolveDirectory(tenantIdentityOrLocalPath string) (string, error)
}
type DirCache interface {
	DirResolver
	DeleteEntry(directory string)
	GetEntry(directory string) (*LogDirCacheEntry, bool)
	FindMassifFiles(directory string) error
	ReadMassifStart(filepath string) (MassifStart, string, error)
	ReadMassif(directory string, massifIndex uint64) (MassifContext, error)
	Open(fileName string) (io.ReadCloser, error)
	Options() LocalReaderOptions
}

type LocalMassifReader struct {
	log logger.Logger
	// cache of previously read material, this is typically shared with a LocalSealReader instance
	cache *LogDirCache
}

type LocalReaderOptions struct {
	noGetRootSupport bool

	requireMassifHeight uint8

	// The following options are only relevant to reader implementations which provide for local file access
	massifDirLister DirLister
	sealDirLister   DirLister

	// readers which operate on a local replica of multiple tenant logs specify the root of the replica using this option
	// The paths under this location match the path schema used by datatrails for the cloud storage of tenant logs
	replicaDir string

	// The following options are only relevant when the reader is configured to read seals
	codec cbor.CBORCodec
}

type LocalReaderOption func(*LocalReaderOptions)

// WithoutGetRootSupport disables the random access map for the peak stack.
// This typically should only be set by log builders
func WithoutLocalGetRootSupport() LocalReaderOption {
	return func(opts *LocalReaderOptions) {
		opts.noGetRootSupport = true
	}
}

func WithLocalRequireMassifHeight(massifHeight uint8) LocalReaderOption {
	return func(opts *LocalReaderOptions) {
		opts.requireMassifHeight = massifHeight
	}
}

// WithLocalReplicaDir specifies a directory under which a local filesystem
// replica of one or more tenant logs is maintained. The filesystem structure
// matches the remote log path structure
func WithLocalReplicaDir(replicaDir string) LocalReaderOption {
	return func(o *LocalReaderOptions) {
		o.replicaDir = replicaDir
	}
}

func WithLocalMassifLister(dirLister DirLister) LocalReaderOption {
	return func(o *LocalReaderOptions) {
		o.massifDirLister = dirLister
	}
}
func WithLocalSealLister(dirLister DirLister) LocalReaderOption {
	return func(o *LocalReaderOptions) {
		o.sealDirLister = dirLister
	}
}
func WithLocalCBORCodec(codec cbor.CBORCodec) LocalReaderOption {
	return func(o *LocalReaderOptions) {
		o.codec = codec
	}
}

func WithSealLister(dir string, dirLister DirLister) LocalReaderOption {
	return func(o *LocalReaderOptions) {
		o.sealDirLister = dirLister
	}
}

func NewLocalReader(
	log logger.Logger, opener Opener, opts ...LocalReaderOption,
) LocalMassifReader {
	r := LocalMassifReader{
		log:   log,
		cache: NewLogDirCache(log, opener, opts...),
	}
	return r
}

func (r *LocalMassifReader) InReplicaMode() bool {
	return r.cache.opts.replicaDir == ""
}

func (r *LocalMassifReader) ReadMassifStart(logfile string) (MassifStart, string, error) {
	return r.cache.ReadMassifStart(logfile)
}

func (r *LocalMassifReader) GetMassif(
	ctx context.Context, tenantIdentityOrLocalPath string, massifIndex uint64,
	opts ...ReaderOption,
) (MassifContext, error) {

	// short circuit direct match, regardless of mode, to support explicit paths
	dirEntry, ok := r.cache.GetEntry(tenantIdentityOrLocalPath)
	if ok {
		return dirEntry.ReadMassif(r.cache, massifIndex)
	}

	directory, err := r.cache.ResolveDirectory(tenantIdentityOrLocalPath)
	if err != nil {
		return MassifContext{}, err
	}

	return r.cache.ReadMassif(directory, massifIndex)
}

func (r *LocalMassifReader) GetHeadMassif(
	ctx context.Context, tenantIdentityOrLocalPath string,
	opts ...ReaderOption,
) (MassifContext, error) {

	dirEntry, ok := r.cache.GetEntry(tenantIdentityOrLocalPath)
	if ok {
		return dirEntry.ReadMassif(r.cache, uint64(dirEntry.HeadMassifIndex))
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
	return dirEntry.ReadMassif(r.cache, uint64(dirEntry.HeadMassifIndex))
}

func (r *LocalMassifReader) GetFirstMassif(
	ctx context.Context, tenantIdentityOrLocalPath string,
	opts ...ReaderOption,
) (MassifContext, error) {
	dirEntry, ok := r.cache.GetEntry(tenantIdentityOrLocalPath)
	if ok {
		return dirEntry.ReadMassif(r.cache, uint64(dirEntry.FirstMassifIndex))
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
	return dirEntry.ReadMassif(r.cache, uint64(dirEntry.FirstMassifIndex))
}

// GetLazyContext is an optimization for remote massif readers
// and is therefor not implemented for local massif reader
func (r *LocalMassifReader) GetLazyContext(
	ctx context.Context, tenantIdentity string, which LogicalBlob,
	opts ...ReaderOption,
) (LogBlobContext, uint64, error) {

	return LogBlobContext{}, 0, fmt.Errorf("not implemented for local storage")
}
