package massifs

import (
	"testing"
)

func TestParseMassifPathTenant(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"happy case",
			args{"v1/mmrs/tenant/84e0e9e9-d479-4d4e-9e8c-afc19a8fc185/0/massifs/0000000000000000.log"},
			"84e0e9e9-d479-4d4e-9e8c-afc19a8fc185",
			false,
		},
		{
			"missing prefix",
			args{"d479-4d4e-9e8c-afc19a8fc185/0/massifs/0000000000000000.log"},
			"",
			true,
		},
		{
			"corrupt prefix",
			args{"v1/mmrx/tenanx/84e0e9e9-d479-4d4e-9e8c-afc19a8fc185/0/massifs/0000000000000000.log"},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMassifPathTenant(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMassifPathTenant() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseMassifPathTenant() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseMassifPathNumberExt(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    uint32
		want1   string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			"happy case",
			args{"v1/mmrs/tenant/84e0e9e9-d479-4d4e-9e8c-afc19a8fc185/0/massifs/0000000000000002.log"},
			2,
			"log",
			false,
		},
		{
			"seals happy case",
			args{"v1/mmrs/tenant/84e0e9e9-d479-4d4e-9e8c-afc19a8fc185/0/massifseals/0000000000000002.sth"},
			2,
			"sth",
			false,
		},

		{
			"bad log ext",
			args{"v1/mmrs/tenant/84e0e9e9-d479-4d4e-9e8c-afc19a8fc185/0/massifs/0000000000000002.lxg"},
			0,
			"",
			true,
		},
		{
			"to few parts in base log name",
			args{"v1/mmrs/tenant/84e0e9e9-d479-4d4e-9e8c-afc19a8fc185/0/massifs/0000000000000000"},
			0,
			"",
			true,
		},
		{
			"un parsable log number",
			args{"v1/mmrs/tenant/84e0e9e9-d479-4d4e-9e8c-afc19a8fc185/0/massifs/0000000y00z00000.log"},
			0,
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := ParseMassifPathNumberExt(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMassifPathNumberExt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseMassifPathNumberExt() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("ParseMassifPathNumberExt() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestIsMassifPathLike(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// notice that the "like tests are intended as a simplifying pre-filter
		{"happy case", args{"v1/mmrs/tenant/log"}, true},
		{"negative case 1", args{"v1/mmrs/tenant/lox"}, false},
		{"negative case 1", args{"v1/mxrs/tenant/log"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsMassifPathLike(tt.args.path); got != tt.want {
				t.Errorf("IsMassifPathLike() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsSealPathLike(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// notice that the "like tests are intended as a simplifying pre-filter
		{"happy case", args{"v1/mmrs/tenant/sth"}, true},
		{"negative case 1", args{"v1/mmrs/tenant/sty"}, false},
		{"negative case 1", args{"v1/mxrs/tenant/sth"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSealPathLike(tt.args.path); got != tt.want {
				t.Errorf("IsSealPathLike() = %v, want %v", got, tt.want)
			}
		})
	}
}
