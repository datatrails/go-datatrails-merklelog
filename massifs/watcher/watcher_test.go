package watcher

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type checkWatchConfig func(t *testing.T, cfg WatchConfig)

func TestConfigDefaults(t *testing.T) {
	hourSince := time.Now().Add(-time.Hour)

	type args struct {
		cfg *WatchConfig
	}
	tests := []struct {
		name      string
		args      args
		errPrefix string
		check     checkWatchConfig
	}{
		{
			name: "horizon or since options are required",
			args: args{
				cfg: &WatchConfig{},
			},
			errPrefix: "provide horizon on its own or either of the since",
		},

		{
			name: "poll with since an hour in the past",
			args: args{
				cfg: &WatchConfig{
					Since: hourSince,
				},
			},
			check: func(t *testing.T, cfg WatchConfig) {
				assert.Equal(t, hourSince, cfg.Since)
				assert.Equal(t, time.Second, cfg.Interval)
				assert.NotEqual(t, "", cfg.IDSince) // should be set to IDTimeHex
			},
		},

		{
			name: "bad hex string for idtimestamp errors",
			args: args{
				cfg: &WatchConfig{
					IDSince: "thisisnothex",
				},
			},
			errPrefix: "encoding/hex: invalid byte",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ConfigDefaults(tt.args.cfg)
			if err != nil {
				if tt.errPrefix == "" {
					t.Errorf("NewWatchConfig() unexpected error = %v", err)
				}
				if !strings.HasPrefix(err.Error(), tt.errPrefix) {
					t.Errorf("NewWatchConfig() unexpected error = %v, expected prefix: %s", err, tt.errPrefix)
				}
			} else {
				if tt.errPrefix != "" {
					t.Errorf("NewWatchConfig() expected error prefix = %s", tt.errPrefix)
				}
			}
			if tt.check != nil {
				tt.check(t, *tt.args.cfg)
			}
		})
	}
}
