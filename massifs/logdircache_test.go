package massifs

import (
	"testing"

	"github.com/datatrails/go-datatrails-common/logger"
)

func TestLogDirCache_ResolveDirectory(t *testing.T) {
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
		// TODO: Add test cases illustrating the different scenarios for tenant
		// identity vs local path and whether or not the replicaDir option was
		// set.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &LogDirCache{
				log:     tt.fields.log,
				opts:    tt.fields.opts,
				entries: tt.fields.entries,
				opener:  tt.fields.opener,
			}
			got, err := c.ResolveDirectory(tt.args.tenantIdentityOrLocalPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LogDirCache.ResolveDirectory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("LogDirCache.ResolveDirectory() = %v, want %v", got, tt.want)
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
			if got := TenantReplicaDir(tt.args.replicaDir, tt.args.tenantIdentity); got != tt.want {
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
			if got := TenantReplicaPath(tt.args.tenantIdentity); got != tt.want {
				t.Errorf("TenantReplicaPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
