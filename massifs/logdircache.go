package massifs

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/datatrails/go-datatrails-common/logger"
)

var (
	ErrLogFileNoMagic                = errors.New("the file is not recognized as a massif")
	ErrLogFileBadHeader              = errors.New("a massif file header was to short or badly formed")
	ErrLogFileMassifHeightHeader     = errors.New("the massif height in the header did not match the expected height")
	ErrLogFileDuplicateMassifIndices = errors.New("log files with the same massif index found in a single directory")
	ErrLogFileMassifNotFound         = errors.New("a log file corresponding to the massif index was not found")
	ErrLogFileSealNotFound           = errors.New("a log seal corresponding to the massif index was not found")
	ErrMassifDirListerNotProvided    = errors.New("the reader option providing a massif directory lister was not provided")
	ErrSealDirListerNotProvided      = errors.New("the reader option providing a massif seal directory lister was not provided")
	ErrAddSealExists                 = errors.New("attempt to add a sealed stated entry that already exists")
	ErrLogMassifHeightNotProvided    = errors.New("a consistent massif height must be specified for all files logically part of the same log")
)

type DirLister interface {
	// ListFiles returns list of absolute paths
	// to files (not subdirectories) in a directory
	ListFiles(string) ([]string, error)
}

type Opener interface {
	Open(string) (io.ReadCloser, error)
}

type DirCache interface {
	DirResolver
	ReplaceMassif(logfile string, mc *MassifContext) error
	ReplaceSeal(sealFilename string, massifIndex uint32, sealedState *SealedState) error
	DeleteEntry(directory string)
	GetEntry(directory string) (*LogDirCacheEntry, bool)
	FindMassifFiles(directory string) error
	ReadMassifStart(filepath string) (MassifStart, string, error)
	ReadMassif(directory string, massifIndex uint64) (*MassifContext, error)
	ReadSeal(directory string, massifIndex uint64) (*SealedState, error)
	Open(fileName string) (io.ReadCloser, error)
	Options() DirCacheOptions
}

type DirCacheOptions struct {
	ReaderOptions

	// The following options are only relevant to reader implementations which provide for local file access
	massifDirLister DirLister
	sealDirLister   DirLister

	// readers which operate on a local replica of multiple tenant logs specify the root of the replica using this option
	// The paths under this location match the path schema used by datatrails for the cloud storage of tenant logs
	replicaDir string
}

// NewLogDirCacheOptions creates a new DirCacheOptions object with the provided options
// Typically, this is used for mocking as the options values are private
func NewLogDirCacheOptions(baseOpts ReaderOptions, opts ...DirCacheOption) DirCacheOptions {

	options := DirCacheOptions{
		ReaderOptions: ReaderOptionsCopy(baseOpts),
	}
	for _, o := range opts {
		o(&options)
	}
	return options
}

type DirCacheOption func(*DirCacheOptions)

// WithoutGetRootSupport disables the random access map for the peak stack.
// This typically should only be set by log builders
func WithoutLocalGetRootSupport() DirCacheOption {
	return func(opts *DirCacheOptions) {
		opts.noGetRootSupport = true
	}
}

func WithReaderOption(o ReaderOption) DirCacheOption {
	return func(opts *DirCacheOptions) {
		o(&opts.ReaderOptions)
	}
}

// WithDirCacheReplicaDir specifies a directory under which a local filesystem
// replica of one or more tenant logs is maintained. The filesystem structure
// matches the remote log path structure
func WithDirCacheReplicaDir(replicaDir string) DirCacheOption {
	return func(o *DirCacheOptions) {
		o.replicaDir = replicaDir
	}
}

func WithDirCacheMassifLister(dirLister DirLister) DirCacheOption {
	return func(o *DirCacheOptions) {
		o.massifDirLister = dirLister
	}
}

func WithDirCacheSealLister(dirLister DirLister) DirCacheOption {
	return func(o *DirCacheOptions) {
		o.sealDirLister = dirLister
	}
}

type LogDirCacheEntry struct {
	LogDirPath       string
	FirstMassifIndex uint32
	HeadMassifIndex  uint32
	FirstSealIndex   uint32
	HeadSealIndex    uint32
	MassifStarts     map[string]MassifStart
	Massifs          map[string]*MassifContext
	Seals            map[string]*SealedState
	MassifPaths      map[uint64]string
	SealPaths        map[uint64]string
}

func NewLogDirCacheEntry(directory string) *LogDirCacheEntry {
	return &LogDirCacheEntry{
		LogDirPath:       directory,
		FirstMassifIndex: ^uint32(0),
		FirstSealIndex:   ^uint32(0),
		MassifStarts:     make(map[string]MassifStart),
		Massifs:          make(map[string]*MassifContext),
		Seals:            make(map[string]*SealedState),
		MassifPaths:      make(map[uint64]string),
		SealPaths:        make(map[uint64]string),
	}
}

// LogDirCache caches the results of scanning a directory for a specific kind of
// merkle log file.  massif .log files and seal .sth files are both supported A
// single cache entry applies all supported file types. A cache may, and should
// be, shared between multiple reader instances, however note that the
// implementation assumes single threaded access. it is not go routine safe.
type LogDirCache struct {
	log     logger.Logger
	opts    DirCacheOptions
	entries map[string]*LogDirCacheEntry
	opener  Opener
}

func NewLogDirCache(log logger.Logger, opener Opener, opts ...DirCacheOption) (*LogDirCache, error) {
	c := &LogDirCache{
		log:     log,
		entries: make(map[string]*LogDirCacheEntry),
		opener:  opener,
	}

	for _, o := range opts {
		o(&c.opts)
	}
	if c.opts.massifHeight == 0 {
		return nil, ErrLogMassifHeightNotProvided
	}
	return c, nil
}

func (c *LogDirCache) Options() DirCacheOptions {
	return c.opts
}

func (c *LogDirCache) Open(filePath string) (io.ReadCloser, error) {
	return c.opener.Open(filePath)
}

// DeleteEntry removes the cached results for a single directory
func (c *LogDirCache) DeleteEntry(directory string) {
	delete(c.entries, directory)
}

// GetEntry returns an existing entry and true or nil and false if the directory does not exist
func (c *LogDirCache) GetEntry(directory string) (*LogDirCacheEntry, bool) {
	d, ok := c.entries[directory]
	return d, ok
}

// ReplaceMassif adds the massif context to the appropriate directory entry
//
// The caller is responsible for providing a fully initialized context (as returned by ReadMassif).
// It is safe for the caller to defer reading the data, provided it guarantees
// to read and finalize the context before any call to ReadMassif.
// The cache entry setup (max / min massif index etc) depends only on
// information in the Start of the context.
func (c *LogDirCache) ReplaceMassif(logfile string, mc *MassifContext) error {
	dirEntry := c.getDirEntry(filepath.Dir(logfile))
	err := dirEntry.setMassifStart(c.opts, logfile, mc.Start)
	if err != nil {
		return err
	}

	// note: it is the callers choice on whether to create the peak stack map
	dirEntry.Massifs[logfile] = mc
	return nil
}

func (c *LogDirCache) ReplaceSeal(sealFilename string, massifIndex uint32, sealedState *SealedState) error {
	dirEntry := c.getDirEntry(filepath.Dir(sealFilename))
	err := dirEntry.setSeal(massifIndex, sealFilename, sealedState)
	if err != nil {
		return err
	}

	dirEntry.Seals[sealFilename] = sealedState
	return nil
}

// FindLogFiles finds and reads massif files from the provided directory
func (c *LogDirCache) FindMassifFiles(directory string) error {

	dirEntry := c.getDirEntry(directory)

	// read all the entries in our log dir
	entries, err := c.opts.massifDirLister.ListFiles(directory)
	if err != nil {
		return err
	}

	// for each entry we read the header (first 32 bytes)
	// and do rough checks if the header looks like it's from a valid log
	for _, filepath := range entries {
		_, err := dirEntry.ReadMassifStart(c, filepath)
		if err != nil && !errors.Is(err, ErrLogFileNoMagic) {
			return err
		}
	}
	return nil
}

// FindSealFiles finds and reads massif seal files from the provided directory
func (c *LogDirCache) FindSealFiles(directory string) error {

	dirEntry := c.getDirEntry(directory)

	// read all the entries in our log dir
	entries, err := c.opts.sealDirLister.ListFiles(directory)
	if err != nil {
		return err
	}

	// for each entry we read the header (first 32 bytes)
	// and do rough checks if the header looks like it's from a valid log
	for _, filepath := range entries {

		_, err := dirEntry.ReadSeal(c, filepath)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReadMassifStart reads and caches the start header for the log file
// The directory for the cache entry is established from the logfile name
// The established directory for the cache entry is returned
func (c *LogDirCache) ReadMassifStart(logfile string) (MassifStart, string, error) {
	dirEntry := c.getDirEntry(filepath.Dir(logfile))
	ms, err := dirEntry.ReadMassifStart(c, logfile)
	return ms, dirEntry.LogDirPath, err
}

// ReadMassif reads the massif, identified by its index, from the provided
// directory A directory cache entry is established for directory if it has not
// previously been scanned, otherwise, the previous scan is re-used.
func (c *LogDirCache) ReadMassif(directory string, massifIndex uint64) (*MassifContext, error) {

	// If we haven't scanned this directory before, scan it now.
	_, ok := c.entries[directory]
	if !ok {
		err := c.FindMassifFiles(directory)
		if err != nil {
			return nil, err
		}
	}
	// If the scan did not find the massif path, ReadMassif will error
	dirEntry := c.getDirEntry(directory)
	return dirEntry.ReadMassif(c, massifIndex)
}

func (c *LogDirCache) ReadSeal(directory string, massifIndex uint64) (*SealedState, error) {

	// If we haven't scanned this directory before, scan it now.
	_, ok := c.entries[directory]
	if !ok {
		err := c.FindSealFiles(directory)
		if err != nil {
			return nil, err
		}
	}
	// If the scan did not find the massif path, ReadMassif will error
	dirEntry := c.getDirEntry(directory)
	return dirEntry.GetSeal(c, massifIndex)
}

// ResolveDirectory resolves a string which may be either a tenant identity or a local path.
//
// If we are regular file or directory mode, this requires that the provided
// value is a directory or a file.
//
// In replica mode, derive the path using the tenantIdentity, and the massif
// blob path schema, and similarly require an existing file path.
//
// Returns
//   - a directory path
func (c *LogDirCache) ResolveDirectory(tenantIdentityOrLocalPath string) (string, error) {
	var err error
	var directory string
	// If we are regular file or directory mode, require that the provided value is a directory or a file
	if c.opts.replicaDir == "" {
		directory, err = dirFromFilepath(tenantIdentityOrLocalPath)
		if err != nil {
			return "", err
		}
		return directory, nil
	}

	directory = tenantReplicaDir(c.opts.replicaDir, tenantIdentityOrLocalPath)
	fi, err := pathInfo(directory)
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrPathIsNotDir, directory)
	}
	return directory, nil
}

// getDirEntry returns the entry for directory or creates a new one and establishes it in the cache
func (c *LogDirCache) getDirEntry(directory string) *LogDirCacheEntry {
	var ok bool
	var dirEntry *LogDirCacheEntry

	// If we have an entry for this directory, re-use it, otherwise create a new one
	if dirEntry, ok = c.entries[directory]; !ok {
		dirEntry = NewLogDirCacheEntry(directory)
		c.entries[directory] = dirEntry
	}
	return dirEntry
}

// ReadMassif returns a MassifContext for the provided massifIndex
// If it has been previously read and prepared, an independent copy of the previously read MassifContext is returned.
func (d *LogDirCacheEntry) ReadMassif(c DirCache, massifIndex uint64) (*MassifContext, error) {
	var err error
	var ok bool
	var fileName string
	var cached *MassifContext
	// check if massif with particular index was found
	if fileName, ok = d.MassifPaths[massifIndex]; !ok {
		return nil, fmt.Errorf("%v: %d", ErrLogFileMassifNotFound, massifIndex)
	}

	if cached, ok = d.Massifs[fileName]; ok {
		return cached, nil
	}

	cached = &MassifContext{}

	reader, err := c.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	// read the data from a file
	cached.Data, err = io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// unmarshal
	err = cached.Start.UnmarshalBinary(cached.Data)
	if err != nil {
		return nil, err
	}

	if !c.Options().noGetRootSupport {
		if err = cached.CreatePeakStackMap(); err != nil {
			return nil, err
		}
	}

	d.Massifs[fileName] = cached

	return cached, nil
}

func (d *LogDirCacheEntry) GetSeal(c DirCache, massifIndex uint64) (*SealedState, error) {
	var ok bool
	var fileName string
	var cached *SealedState
	// check if seal with particular index was found
	if fileName, ok = d.SealPaths[massifIndex]; !ok {
		return nil, fmt.Errorf("%v: %d", ErrLogFileSealNotFound, massifIndex)
	}

	if cached, ok = d.Seals[fileName]; ok {
		return cached, nil
	}
	return nil, fmt.Errorf("%v: %d", ErrLogFileSealNotFound, massifIndex)
}

func (d *LogDirCacheEntry) ReadSeal(c DirCache, fileName string) (*SealedState, error) {
	var err error
	var ok bool
	var cached *SealedState

	if cached, ok = d.Seals[fileName]; ok {
		return cached, nil
	}

	cached = &SealedState{}

	reader, err := c.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	// read the data from a file
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	cachedMessage, unverifiedState, err := DecodeSignedRoot(*c.Options().codec, data)
	if err != nil {
		return nil, err
	}
	cached.MMRState = unverifiedState
	cached.Sign1Message = *cachedMessage

	d.Seals[fileName] = cached
	massifIndex := MassifIndexFromMMRIndex(c.Options().massifHeight, unverifiedState.MMRSize-1)
	d.SealPaths[massifIndex] = fileName

	return cached, nil
}

func (d *LogDirCacheEntry) ReadMassifStart(dirCache *LogDirCache, logfile string) (MassifStart, error) {

	if ms, ok := d.MassifStarts[logfile]; ok {
		return ms, nil
	}

	f, err := dirCache.opener.Open(logfile)
	if err != nil {
		return MassifStart{}, err
	}
	defer f.Close()
	header := make([]byte, 32)

	i, err := f.Read(header)
	if err != nil && !errors.Is(err, io.EOF) {
		return MassifStart{}, err
	}

	// if we read less than 32 bytes we ignore the file completely
	// as it's not a valid log
	if i != 32 {
		return MassifStart{}, ErrLogFileBadHeader
	}

	// unmarshal the header
	ms := MassifStart{}
	err = DecodeMassifStart(&ms, header)
	if err != nil {
		return MassifStart{}, err
	}
	err = d.setMassifStart(dirCache.opts, logfile, ms)
	if err != nil {
		return MassifStart{}, err
	}

	return ms, nil
}

func (d *LogDirCacheEntry) setMassifStart(opts DirCacheOptions, logfile string, ms MassifStart) error {
	// The type field is currently zero
	if ms.Reserved != 0 {
		return fmt.Errorf("%w: reserved bytes not zero", ErrLogFileNoMagic)
	}
	if ms.Version != 0 {
		return fmt.Errorf("%w: unexpected (or not supported) massif version: %d", ErrLogFileNoMagic, ms.Version)
	}

	// note: we could require the epoch to be 1, but that would interfere with testing
	// same for the massifHeight

	// If the options require a specific massif height check the height we got from the header.
	if opts.massifHeight != 0 && ms.MassifHeight != opts.massifHeight {
		return fmt.Errorf("%w: header=%d, expected=%d", ErrLogFileMassifHeightHeader, ms.MassifHeight, opts.massifHeight)
	}

	// if we already have a log with the same massifIndex, read from a
	// *different file*, we error out as the files in directories are
	// potentially not for the same tenancy - which means the data is not
	// correct
	if cachedLogFile, ok := d.MassifPaths[uint64(ms.MassifIndex)]; ok && cachedLogFile != logfile {
		return fmt.Errorf("%w: %s and %s", ErrLogFileDuplicateMassifIndices, cachedLogFile, logfile)
	}

	// associate filename with the massif index
	d.MassifPaths[uint64(ms.MassifIndex)] = logfile
	d.MassifStarts[logfile] = ms

	// update the head massif index if we have new one
	if ms.MassifIndex > d.HeadMassifIndex {
		d.HeadMassifIndex = ms.MassifIndex
	}

	// update the first massif index if we have new one
	if ms.MassifIndex < d.FirstMassifIndex {
		d.FirstMassifIndex = ms.MassifIndex
	}
	return nil
}

// err := dirEntry.setSeal(c.opts, sealFilename, massifIndex, sealedState)
func (d *LogDirCacheEntry) setSeal(
	massifIndex uint32, sealFilename string, seal *SealedState,
) error {

	// if we already have a log with the same massifIndex, read from a
	// *different file*, we error out as the files in directories are
	// potentially not for the same tenancy - which means the data is not
	// correct
	if cachedSealFile, ok := d.SealPaths[uint64(massifIndex)]; ok && cachedSealFile != sealFilename {
		return fmt.Errorf("%w: %s and %s", ErrLogFileDuplicateMassifIndices, cachedSealFile, sealFilename)
	}

	// associate filename with the massif index
	d.SealPaths[uint64(massifIndex)] = sealFilename
	d.Seals[sealFilename] = seal

	// update the head massif index if we have new one
	if massifIndex > d.HeadSealIndex {
		d.HeadSealIndex = massifIndex
	}

	// update the first massif index if we have new one
	if massifIndex < d.FirstMassifIndex {
		d.FirstSealIndex = massifIndex
	}
	return nil
}

// tenantReplicaPath normalizes a tenantIdentity to conform to our remotes
// storage path schema.
//
// tenantIdentity should be "tenant/UUID", if it's value does not start with
// "tenant/" then the prefix is forcibly added.
func tenantReplicaPath(tenantIdentity string) string {
	// normalize tenant identity
	if !strings.HasPrefix("tenant/", tenantIdentity) {
		tenantIdentity = "tenant/" + tenantIdentity
	}
	return strings.TrimPrefix(TenantMassifPrefix(tenantIdentity), V1MMRPrefix)
}

// tenantReplicaPath normalizes a tenantIdentity to conform to our remotes
// storage path schema and converts the path to use local file system path separators
func tenantReplicaDir(replicaDir, tenantIdentity string) string {
	tenantPath := tenantReplicaPath(tenantIdentity)
	directoryParts := strings.Split(tenantPath, "/")
	if replicaDir != "" {
		directoryParts = append([]string{replicaDir}, directoryParts...)
	}
	return path.Join(directoryParts...)
}

// pathInfo returns the FileInfo for a path
func pathInfo(path string) (fs.FileInfo, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}
	return fi, nil
}

// dirFromFilepath returns an existing directory path derived from path or errors
// if path is a file, the directory is obtained by calling filepath.Dir on the path
func dirFromFilepath(path string) (string, error) {
	orig := path
	fi, err := pathInfo(path)
	if err != nil {
		return "", err
	}
	if fi.IsDir() {
		return path, nil
	}
	// It's a file, derive the directory from the file
	path = filepath.Dir(path)
	fi, err = pathInfo(path)
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("%w: %s derived from %s", ErrPathIsNotDir, path, orig)
	}
	return path, nil
}
