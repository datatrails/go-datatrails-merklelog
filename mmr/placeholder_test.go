package mmr

import "testing"

func TestPlaceHolder(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{
			name: "positive",
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PlaceHolder(); got != tt.want {
				t.Errorf("PlaceHolder() = %v, want %v", got, tt.want)
			}
		})
	}
}
