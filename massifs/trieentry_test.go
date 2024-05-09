package massifs

import (
	"encoding/binary"
	"reflect"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTrieEntryOffset tests:
//
// 1. we get the correct offset for the trie entry at trie index 0
// 2. we get the correct offset for the trie entry at trie index 1
func TestTrieEntryOffset(t *testing.T) {
	type args struct {
		indexStart uint64
		leafIndex  uint64
	}
	tests := []struct {
		name     string
		args     args
		expected uint64
	}{
		{
			name: "first entry",
			args: args{
				indexStart: uint64(100),
				leafIndex:  uint64(0),
			},
			expected: 100,
		},
		{
			name: "second entry",
			args: args{
				indexStart: uint64(100),
				leafIndex:  uint64(1),
			},
			expected: 164,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := TrieEntryOffset(test.args.indexStart, test.args.leafIndex)

			assert.Equal(t, test.expected, actual)
		})
	}
}

// TestNewTrieKey tests:
//
// 1. we can create a trie key and its format is correct.
func TestNewTrieKey(t *testing.T) {
	type args struct {
		logId []byte
		appId []byte
	}
	tests := []struct {
		name     string
		args     args
		expected []byte
	}{
		// These tests simply ensure we are hashing things in the expected
		// order. To update them after changing that order, just run them,
		// capture the new hash and copy it into the test.
		{
			name: "typical datatrails event",
			args: args{
				logId: []byte("tenant/1de5793f-1421-45d8-999e-9513552f8c0b"),
				appId: []byte("assets/9eb98893-e0e3-4c21-99c2-0a88d7b2c2ea/events/c0cd94a9-3489-4957-baf9-cf75d478b53f"),
			},
			expected: []uint8{123, 35, 118, 210, 212, 254, 212, 242, 52, 254, 186, 214, 7, 135, 29, 32, 194, 28, 222, 28, 169, 234, 74, 175, 58, 4, 21, 140, 63, 83, 150, 79},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := NewTrieKey(0, test.args.logId, test.args.appId)

			assert.Equal(t, test.expected, actual)
		})
	}
}

// NewEmptyTrieEntry tests:
//
// 1. we can generate a new empty trie entry.
func TestNewEmptyTrieEntry(t *testing.T) {
	tests := []struct {
		name string
		want []byte
	}{
		{"test empty generated is correct", make([]byte, 64)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewEmptyTrieEntry(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEmptyTrieEntry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetLogIndexEntry(t *testing.T) {

	expectTrieEntryBytes := 64

	// If this fails, then we are likely updating the index format and this test needs updating.
	require.Equal(t, expectTrieEntryBytes, TrieEntryBytes)

	b64clear := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	b64h0 := []byte{0, 0, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0}
	b64h1 := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0}
	b64zero := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	b64one := []byte{1, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	b64two := []byte{2, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}
	b64three := []byte{3, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3}
	b64four := []byte{4, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}
	type args struct {
		logData     []byte
		indexStart  uint64
		leafIndex   uint64
		idTimestamp uint64
		index       []byte
		before      []byte
		after       []byte
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "get trie index 1",
			args: args{
				logData:     slices.Concat(b64h0, b64h1, b64zero, b64one, b64two, b64three, b64four),
				indexStart:  uint64(expectTrieEntryBytes * 2),
				leafIndex:   1, // *trie* index NOT mmrIndex,
				idTimestamp: 0x0102030405060708,
				index:       b64one[:32],
				before:      b64zero[:32],
				after:       b64two[:32],
			},
		},
		{
			name: "get trie index 3",
			args: args{
				logData:     slices.Concat(b64h0, b64h1, b64zero, b64one, b64two, b64three, b64four),
				indexStart:  uint64(expectTrieEntryBytes * 2),
				leafIndex:   3, // *trie* index NOT mmrIndex
				idTimestamp: 0x0102030405060708,
				index:       b64three[:32],
				before:      b64two[:32],
				after:       b64four[:32],
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First test that we get our expected pre-filled data
			got := GetTrieKey(tt.args.logData, tt.args.indexStart, tt.args.leafIndex)
			assert.Equal(t, got, tt.args.index)
			// Check that we can set it to zero without corrupting entries to either side
			if tt.args.before == nil && tt.args.after == nil {
				return
			}
			SetTrieEntry(tt.args.logData, tt.args.indexStart, tt.args.leafIndex, tt.args.idTimestamp, b64clear)

			gotBefore := GetTrieKey(tt.args.logData, tt.args.indexStart, tt.args.leafIndex-1)
			assert.Equal(t, tt.args.before, gotBefore)
			gotAfter := GetTrieKey(tt.args.logData, tt.args.indexStart, tt.args.leafIndex+1)
			assert.Equal(t, tt.args.after, gotAfter)

			// check we get back the zeros we set

			got = GetTrieKey(tt.args.logData, tt.args.indexStart, tt.args.leafIndex)
			assert.Equal(t, b64clear[:32], got)

			gotId := GetIdtimestamp(tt.args.logData, tt.args.indexStart, tt.args.leafIndex)
			assert.Equal(t, tt.args.idTimestamp, binary.BigEndian.Uint64(gotId))
		})
	}
}
