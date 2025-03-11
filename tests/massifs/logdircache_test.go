package massifs

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/datatrails/go-datatrails-common/logger"
	"github.com/datatrails/go-datatrails-merklelog/massifs"
	"github.com/datatrails/go-datatrails-merklelog/massifs/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewLogDirCacheEntry(t *testing.T) {

	logger.New("TestNewLocalReader")
	defer logger.OnExit()

	dl := mocks.NewDirLister(t)
	dl.On("ListFiles", mock.Anything).Return(
		func(name string) ([]string, error) {
			switch name {
			case "/foo/bar":
				return []string{"/foo/bar/log.log"}, nil
			case "/log/massif":
				return []string{"/log/massif/0.log"}, nil
			case "/same/log":
				return []string{"/same/log/0.log", "/same/log/1.log"}, nil
			case "/logs/invalid/":
				return []string{"/logs/invalid/0.log", "/logs/invalid/invalid.log"}, nil
			case "/logs/short":
				return []string{"/logs/short/0.log", "/logs/short/1.log"}, nil
			case "/logs/valid":
				return []string{"/logs/valid/0.log", "/logs/valid/1.log"}, nil
			case "/logs/valid3":
				return []string{"/logs/valid3/0.log", "/logs/valid3/1.log", "/logs/valid3/255.log"}, nil
			default:
				return []string{}, nil
			}
		},
	)

	// this mock returns headers of logfiles
	// signigicant bytes we use in test are 27 for mmr height
	// and last 4 (28-32) for mmr index
	op := mocks.NewOpener(t)
	op.On("Open", mock.Anything).Return(
		func(name string) (io.ReadCloser, error) {
			switch name {
			case "/foo/bar/log.log":
				return nil, fmt.Errorf("bad file log.log")
			case "/log/massif/0.log":
				b, _ := hex.DecodeString("000000000000000090757516a9086b0000000000000000000000010e00000000")
				return io.NopCloser(bytes.NewReader(b)), nil
			case "/same/log/0.log":
				b, _ := hex.DecodeString("000000000000000090757516a9086b0000000000000000000000010e00000000")
				return io.NopCloser(bytes.NewReader(b)), nil
			case "/same/log/1.log":
				b, _ := hex.DecodeString("000000000000000090757515a9086b0000000000000000000000010e00000000")
				return io.NopCloser(bytes.NewReader(b)), nil
			case "/logs/invalid/0.log":
				b, _ := hex.DecodeString("000000000000000090757515a9086b0000000000000000000000010e00000000")
				return io.NopCloser(bytes.NewReader(b)), nil
			case "/logs/invalid/invalid.log":
				b, _ := hex.DecodeString("000000000000000090757515a9086b0000000000000000000000010f00000000")
				return io.NopCloser(bytes.NewReader(b)), nil
			case "/logs/short/0.log":
				b, _ := hex.DecodeString("000000000000000090757515a9086b0000000000000000000000010e00000000")
				return io.NopCloser(bytes.NewReader(b)), nil
			case "/logs/short/1.log":
				b, _ := hex.DecodeString("00000000000000009075")
				return io.NopCloser(bytes.NewReader(b)), nil
			case "/logs/valid/0.log":
				b, _ := hex.DecodeString("000000000000000090757515a9086b0000000000000000000000010e00000000")
				return io.NopCloser(bytes.NewReader(b)), nil
			case "/logs/valid/1.log":
				b, _ := hex.DecodeString("000000000000000090757515a9086b0000000000000000000000010e00000007")
				return io.NopCloser(bytes.NewReader(b)), nil
			case "/logs/valid3/0.log":
				b, _ := hex.DecodeString("000000000000000090757515a9086b0000000000000000000000010e00000000")
				return io.NopCloser(bytes.NewReader(b)), nil
			case "/logs/valid3/1.log":
				b, _ := hex.DecodeString("000000000000000090757515a9086b0000000000000000000000010e00000007")
				return io.NopCloser(bytes.NewReader(b)), nil
			case "/logs/valid3/255.log":
				b, _ := hex.DecodeString("000000000000000090757515a9086b0000000000000000000000010e000000ff")
				return io.NopCloser(bytes.NewReader(b)), nil
			default:
				return nil, nil
			}
		},
	)

	type args struct {
		directory string
	}
	tests := []struct {
		name          string
		opts          []massifs.DirCacheOption
		opener        massifs.Opener
		dirlister     massifs.DirLister
		logs          string
		isdir         bool
		outcome       map[uint64]string
		wantErr       error
		wantErrPrefix string
	}{
		{
			name:          "fail on bad file",
			opts:          []massifs.DirCacheOption{massifs.WithReaderOption(massifs.WithMassifHeight(14))},
			opener:        op,
			dirlister:     dl,
			wantErrPrefix: "bad file log.log",
			isdir:         false,
			logs:          "/foo/bar",
		},

		{
			name:      "log 0 valid",
			opts:      []massifs.DirCacheOption{massifs.WithReaderOption(massifs.WithMassifHeight(14))},
			opener:    op,
			dirlister: dl,
			logs:      "/log/massif",
			isdir:     false,
			outcome:   map[uint64]string{0: "/log/massif/0.log"},
		},
		{
			name:      "valid + invalid height not default",
			opts:      []massifs.DirCacheOption{massifs.WithReaderOption(massifs.WithMassifHeight(14))},
			opener:    op,
			dirlister: dl,
			wantErr:   massifs.ErrLogFileMassifHeightHeader,
			logs:      "/logs/invalid/",
			isdir:     true,
			outcome:   map[uint64]string{0: "/logs/invalid/0.log"},
		},
		{
			name:      "valid + short file",
			opts:      []massifs.DirCacheOption{massifs.WithReaderOption(massifs.WithMassifHeight(14))},
			opener:    op,
			dirlister: dl,
			logs:      "/logs/short",
			isdir:     true,
			outcome:   map[uint64]string{0: "/logs/short/0.log"},
		},
		{
			name:      "two valid",
			opts:      []massifs.DirCacheOption{massifs.WithReaderOption(massifs.WithMassifHeight(14))},
			opener:    op,
			dirlister: dl,
			logs:      "/logs/valid",
			isdir:     true,
			outcome: map[uint64]string{
				0: "/logs/valid/0.log",
				7: "/logs/valid/1.log",
			},
		},
		{
			name:      "three valid",
			opts:      []massifs.DirCacheOption{massifs.WithReaderOption(massifs.WithMassifHeight(14))},
			opener:    op,
			dirlister: dl,
			logs:      "/logs/valid3",
			isdir:     true,
			outcome: map[uint64]string{
				0:   "/logs/valid3/0.log",
				7:   "/logs/valid3/1.log",
				255: "/logs/valid3/255.log",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			cache, err := massifs.NewLogDirCache(logger.Sugar, op, append(tt.opts, massifs.WithDirCacheMassifLister(dl))...)
			assert.NoError(t, err)

			err = cache.FindMassifFiles(tt.logs)
			if tt.wantErr != nil {
				assert.NotNil(t, err, "expected error got nil")
				assert.ErrorIs(t, err, tt.wantErr)
			} else if tt.wantErrPrefix != "" {
				assert.NotNil(t, err, "expected error got nil")
				assert.True(t, strings.HasPrefix(err.Error(), tt.wantErrPrefix))
			} else {
				assert.Nil(t, err, "unexpected error")
				e, ok := cache.GetEntry(tt.logs)
				assert.True(t, ok)
				dirEntry := e.(*massifs.LogDirCacheEntry)
				assert.Equal(t, tt.outcome, dirEntry.MassifPaths)
			}
		})
	}
}
