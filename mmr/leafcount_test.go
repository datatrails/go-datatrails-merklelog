package mmr

import "testing"

func TestLeafCount(t *testing.T) {
	type args struct {
		size uint64
	}
	tests := []struct {
		name   string
		args   args
		leaves uint64
	}{
		// 3              14
		//              /    \
		//             /      \
		//            /        \
		//           /          \
		// 2        6            13           21
		//        /   \        /    \
		// 1     2     5      9     12     17     20     24
		//      / \   / \    / \   /  \   /  \
		// 0   0   1 3   4  7   8 10  11 15  16 18  19 22  23   25

		{
			name: "size 15 has 8 leaves",
			args: args{
				size: 14 + 1,
			},
			leaves: 8,
		},
		{
			name: "size 11 has 7 leaves",
			args: args{
				size: 10 + 1,
			},
			leaves: 7,
		},

		{
			name: "invalid size 12 has 7 leaves",
			args: args{
				size: 11 + 1,
			},
			leaves: 7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LeafCount(tt.args.size); got != tt.leaves {
				t.Errorf("LeafCount() = %v, want %v", got, tt.leaves)
			}
		})
	}
}
