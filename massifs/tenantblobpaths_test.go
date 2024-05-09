package massifs

import (
	"testing"
)

func TestTenantMassifPrefix(t *testing.T) {
	type args struct {
		tenantIdentity string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{args: args{"tenant/1234"}, want: "v1/mmrs/tenant/1234/0/massifs/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TenantMassifPrefix(tt.args.tenantIdentity); got != tt.want {
				t.Errorf("TenantMassifPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMassifPrefixForTenantUUID(t *testing.T) {
	type args struct {
		tenantUUID string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{args: args{"1234"}, want: "v1/mmrs/tenant/1234/0/massifs/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MassifPrefixForTenantUUID(tt.args.tenantUUID); got != tt.want {
				t.Errorf("MassifPrefixForTenantUUID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTenantMassifSignedRootsPrefix(t *testing.T) {
	type args struct {
		tenantIdentity string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{args: args{"tenant/1234"}, want: "v1/mmrs/tenant/1234/0/massifseals/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TenantMassifSignedRootsPrefix(tt.args.tenantIdentity); got != tt.want {
				t.Errorf("TenantMassifSignedRootsPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTenantMassifBlobPath(t *testing.T) {
	type args struct {
		tenantIdentity string
		number         uint64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{args: args{"tenant/1234", 1}, want: "v1/mmrs/tenant/1234/0/massifs/0000000000000001.log"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TenantMassifBlobPath(tt.args.tenantIdentity, tt.args.number); got != tt.want {
				t.Errorf("TenantMassifBlobPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTenantMassifSignedRootPath(t *testing.T) {
	type args struct {
		tenantIdentity string
		massifIndex    uint32
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{args: args{"tenant/1234", 1}, want: "v1/mmrs/tenant/1234/0/massifseals/0000000000000001.sth"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TenantMassifSignedRootPath(tt.args.tenantIdentity, tt.args.massifIndex); got != tt.want {
				t.Errorf("TenantMassifSignedRootPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
