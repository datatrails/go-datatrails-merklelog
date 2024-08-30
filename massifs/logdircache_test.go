package massifs

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/datatrails/go-datatrails-common/logger"
	"github.com/stretchr/testify/require"
)

func TestLogDirCache_ResolveMassifDir(t *testing.T) {

	tmpDir := t.TempDir()

	tmpPath := func(relativePath string) string {
		return filepath.Join(tmpDir, relativePath)
	}

	createTmpPath := func(relativePath string) string {
		dirPath := tmpPath(relativePath)
		os.MkdirAll(dirPath, 0755)
		return dirPath
	}

	createTmpFile := func(relativePath string) string {

		filePath := tmpPath(relativePath)
		os.MkdirAll(filepath.Dir(filePath), 0755)

		f, err := os.Create(filePath)
		require.NoError(t, err)
		f.Close()
		return filePath
	}

	tenant0 := "tenant/1234"
	tenantInvalid := "tenant/unknown"

	// The existence of this directory should not affect the the explicit file path mode test cases
	// Failure of this expectation is part of what this test should catch.
	createTmpPath(fmt.Sprintf("%s/0/massifs", tenant0))

	type fields struct {
		replicaDir           string
		explicitFilePathMode bool
	}
	type args struct {
		identifier string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{

		{
			name:    "in replica dir mode, a tenant id for an non existing tenant replica dir should fail",
			fields:  fields{explicitFilePathMode: false, replicaDir: tmpDir},
			args:    args{tenantInvalid},
			wantErr: true,
		},

		{
			name:   "in replica dir mode, a tenant id for an existing tenant replica dir should succeed",
			fields: fields{explicitFilePathMode: false, replicaDir: tmpDir},
			args:   args{tenant0},
			want:   tmpPath(fmt.Sprintf("%s/0/massifs", tenant0)),
		},

		{
			name:    "explicit file non-existing file should fail (even if it is tenant id like)",
			fields:  fields{explicitFilePathMode: true},
			args:    args{tenant0},
			wantErr: true,
		},

		{
			name: "explicit file non-existing file should fail (even if it is tenant id like)",
			// And even if that tenant id corresponds to a replica director
			fields:  fields{explicitFilePathMode: true},
			args:    args{tenant0},
			wantErr: true,
		},

		{
			name:   "explicit file existing file should return the directory containing the file",
			fields: fields{explicitFilePathMode: true},
			args:   args{createTmpFile("mylog/0000000000000000.log")},
			want:   tmpPath("mylog"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := DirCacheOptions{
				replicaDir:           tt.fields.replicaDir,
				explicitFilePathMode: tt.fields.explicitFilePathMode,
			}
			c := &LogDirCache{
				opts: opts,
			}
			got, err := c.ResolveMassifDir(tt.args.identifier)
			if (err != nil) != tt.wantErr {
				t.Errorf("LogDirCache.ResolveMassifDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("LogDirCache.ResolveMassifDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTenantReplicaDir(t *testing.T) {
	type args struct {
		replicaDir     string
		tenantIdentity string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		/* TODO: fix or remove support
		{
			"local file reference",
			args{"replicadir", "tenant/6ea5cd00-c711-3649-6914-7b125928bbb4/0/massifs/0000000000000000.log"},
			"replicadir/tenant/6ea5cd00-c711-3649-6914-7b125928bbb4/0/massifs",
		},*/
		{"empty replica dir", args{"", "1234"}, "tenant/1234/0/massifs"},
		{"full tenant id provided", args{"replicadir", "tenant/1234"}, "replicadir/tenant/1234/0/massifs"},
		{"tenant uuid provided", args{"replicadir", "1234"}, "replicadir/tenant/1234/0/massifs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TenantMassifReplicaDir(tt.args.replicaDir, tt.args.tenantIdentity); got != tt.want {
				t.Errorf("TenantReplicaDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTenantReplicaPath(t *testing.T) {
	type args struct {
		tenantIdentity string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"tenant identity provided", args{"tenant/1234"}, "tenant/1234/0/massifs"},
		{"tenant uuid provided", args{"1234"}, "tenant/1234/0/massifs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TenantMassifReplicaPath(tt.args.tenantIdentity); got != tt.want {
				t.Errorf("TenantReplicaPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogDirCache_ResolveSealDir(t *testing.T) {
	type fields struct {
		log     logger.Logger
		opts    DirCacheOptions
		entries map[string]*LogDirCacheEntry
		opener  Opener
	}
	type args struct {
		tenantIdentityOrLocalPath string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &LogDirCache{
				log:     tt.fields.log,
				opts:    tt.fields.opts,
				entries: tt.fields.entries,
				opener:  tt.fields.opener,
			}
			got, err := c.ResolveSealDir(tt.args.tenantIdentityOrLocalPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LogDirCache.ResolveSealDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("LogDirCache.ResolveSealDir() = %v, want %v", got, tt.want)
			}
		})
	}
}
