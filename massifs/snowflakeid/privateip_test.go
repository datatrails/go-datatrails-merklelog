package snowflakeid

import (
	"fmt"
	"testing"
)

func Test_workerID(t *testing.T) {
	tests := []struct {
		optional string
		cfg      Config
		want     uint16
		want1    int
		wantErr  bool
	}{
		{"", Config{WorkerCIDR: "0.0.0.0/16", PodIP: "10.2.3.4"}, 3*(1<<8) + 4, 8, false},
		{"", Config{WorkerCIDR: "0.0.0.0/24", PodIP: "10.2.3.4"}, 4, 16, false},
		{"", Config{WorkerCIDR: "0.0.0.0/23", PodIP: "10.2.3.4"}, 1*(1<<8) + 4, 15, false},

		{"err not private ip", Config{WorkerCIDR: "0.0.0.0/24", PodIP: "1.2.3.4"}, 0, 0, true},
		{"err to many ips", Config{WorkerCIDR: "0.0.0.0/8", PodIP: "1.2.3.4"}, 0, 0, true},
		{"err to few ips", Config{WorkerCIDR: "0.0.0.0/25", PodIP: "1.2.3.4"}, 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%scidr=%s,ip=%s", tt.optional, tt.cfg.WorkerCIDR, tt.cfg.PodIP), func(t *testing.T) {
			got, got1, err := workerIDSequenceBits(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("workerID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("workerID id = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("workerID seqLen = %v, want %v", got1, tt.want1)
			}
		})
	}
}
