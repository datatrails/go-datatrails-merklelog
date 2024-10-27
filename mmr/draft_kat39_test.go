package mmr

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tests and KAT data corresponding to the MMRIVER draft

var (
	KAT39CompleteMMRSizes   = []uint64{1, 3, 4, 7, 8, 10, 11, 15, 16, 18, 19, 22, 23, 25, 26, 31, 32, 34, 35, 38, 39}
	KAT39CompleteMMRIndices = []uint64{0, 2, 3, 6, 7, 9, 10, 14, 15, 17, 18, 21, 22, 24, 25, 30, 31, 33, 34, 37, 38}
	KAT39LeafMMRIndices     = []uint64{0, 1, 3, 4, 7, 8, 10, 11, 15, 16, 18, 19, 22, 23, 25, 26, 31, 32, 34, 35, 38}
	KAT39PeakIndices        = map[uint64][]uint64{
		0:  {0},
		2:  {2},
		3:  {2, 3},
		6:  {6},
		7:  {6, 7},
		9:  {6, 9},
		10: {6, 9, 10},
		14: {14},
		15: {14, 15},
		17: {14, 17},
		18: {14, 17, 18},
		21: {14, 21},
		22: {14, 21, 22},
		24: {14, 21, 24},
		25: {14, 21, 24, 25},
		30: {30},
		31: {30, 31},
		33: {30, 33},
		34: {30, 33, 34},
		37: {30, 37},
		38: {30, 37, 38},
	}
	// Note: its just easier all round to maintain these as hex strings and convert to bytes on demand.
	KAT39PeakHashes = map[uint64][]string{
		0:  {"af5570f5a1810b7af78caf4bc70a660f0df51e42baf91d4de5b2328de0e83dfc"},
		2:  {"ad104051c516812ea5874ca3ff06d0258303623d04307c41ec80a7a18b332ef8"},
		3:  {"ad104051c516812ea5874ca3ff06d0258303623d04307c41ec80a7a18b332ef8", "d5688a52d55a02ec4aea5ec1eadfffe1c9e0ee6a4ddbe2377f98326d42dfc975"},
		6:  {"827f3213c1de0d4c6277caccc1eeca325e45dfe2c65adce1943774218db61f88"},
		7:  {"827f3213c1de0d4c6277caccc1eeca325e45dfe2c65adce1943774218db61f88", "a3eb8db89fc5123ccfd49585059f292bc40a1c0d550b860f24f84efb4760fbf2"},
		9:  {"827f3213c1de0d4c6277caccc1eeca325e45dfe2c65adce1943774218db61f88", "b8faf5f748f149b04018491a51334499fd8b6060c42a835f361fa9665562d12d"},
		10: {"827f3213c1de0d4c6277caccc1eeca325e45dfe2c65adce1943774218db61f88", "b8faf5f748f149b04018491a51334499fd8b6060c42a835f361fa9665562d12d", "8d85f8467240628a94819b26bee26e3a9b2804334c63482deacec8d64ab4e1e7"},
		14: {"78b2b4162eb2c58b229288bbcb5b7d97c7a1154eed3161905fb0f180eba6f112"},
		15: {"78b2b4162eb2c58b229288bbcb5b7d97c7a1154eed3161905fb0f180eba6f112", "e66c57014a6156061ae669809ec5d735e484e8fcfd540e110c9b04f84c0b4504"},
		17: {"78b2b4162eb2c58b229288bbcb5b7d97c7a1154eed3161905fb0f180eba6f112", "f4a0db79de0fee128fbe95ecf3509646203909dc447ae911aa29416bf6fcba21"},
		18: {"78b2b4162eb2c58b229288bbcb5b7d97c7a1154eed3161905fb0f180eba6f112", "f4a0db79de0fee128fbe95ecf3509646203909dc447ae911aa29416bf6fcba21", "5bc67471c189d78c76461dcab6141a733bdab3799d1d69e0c419119c92e82b3d"},
		21: {"78b2b4162eb2c58b229288bbcb5b7d97c7a1154eed3161905fb0f180eba6f112", "61b3ff808934301578c9ed7402e3dd7dfe98b630acdf26d1fd2698a3c4a22710"},
		22: {"78b2b4162eb2c58b229288bbcb5b7d97c7a1154eed3161905fb0f180eba6f112", "61b3ff808934301578c9ed7402e3dd7dfe98b630acdf26d1fd2698a3c4a22710", "7a42e3892368f826928202014a6ca95a3d8d846df25088da80018663edf96b1c"},
		24: {"78b2b4162eb2c58b229288bbcb5b7d97c7a1154eed3161905fb0f180eba6f112", "61b3ff808934301578c9ed7402e3dd7dfe98b630acdf26d1fd2698a3c4a22710", "dd7efba5f1824103f1fa820a5c9e6cd90a82cf123d88bd035c7e5da0aba8a9ae"},
		25: {"78b2b4162eb2c58b229288bbcb5b7d97c7a1154eed3161905fb0f180eba6f112", "61b3ff808934301578c9ed7402e3dd7dfe98b630acdf26d1fd2698a3c4a22710", "dd7efba5f1824103f1fa820a5c9e6cd90a82cf123d88bd035c7e5da0aba8a9ae", "561f627b4213258dc8863498bb9b07c904c3c65a78c1a36bca329154d1ded213"},
		30: {"d4fb5649422ff2eaf7b1c0b851585a8cfd14fb08ce11addb30075a96309582a7"},
		31: {"d4fb5649422ff2eaf7b1c0b851585a8cfd14fb08ce11addb30075a96309582a7", "1664a6e0ea12d234b4911d011800bb0f8c1101a0f9a49a91ee6e2493e34d8e7b"},
		33: {"d4fb5649422ff2eaf7b1c0b851585a8cfd14fb08ce11addb30075a96309582a7", "0c9f36783b5929d43c97fe4b170d12137e6950ef1b3a8bd254b15bbacbfdee7f"},
		34: {"d4fb5649422ff2eaf7b1c0b851585a8cfd14fb08ce11addb30075a96309582a7", "0c9f36783b5929d43c97fe4b170d12137e6950ef1b3a8bd254b15bbacbfdee7f", "4d75f61869104baa4ccff5be73311be9bdd6cc31779301dfc699479403c8a786"},
		37: {"d4fb5649422ff2eaf7b1c0b851585a8cfd14fb08ce11addb30075a96309582a7", "6a169105dcc487dbbae5747a0fd9b1d33a40320cf91cf9a323579139e7ff72aa"},
		38: {"d4fb5649422ff2eaf7b1c0b851585a8cfd14fb08ce11addb30075a96309582a7", "6a169105dcc487dbbae5747a0fd9b1d33a40320cf91cf9a323579139e7ff72aa", "e9a5f5201eb3c3c856e0a224527af5ac7eb1767fb1aff9bd53ba41a60cde9785"},
	}

	KAT39Leaves = []string{
		"af5570f5a1810b7af78caf4bc70a660f0df51e42baf91d4de5b2328de0e83dfc",
		"cd2662154e6d76b2b2b92e70c0cac3ccf534f9b74eb5b89819ec509083d00a50",
		"d5688a52d55a02ec4aea5ec1eadfffe1c9e0ee6a4ddbe2377f98326d42dfc975",
		"8005f02d43fa06e7d0585fb64c961d57e318b27a145c857bcd3a6bdb413ff7fc",
		"a3eb8db89fc5123ccfd49585059f292bc40a1c0d550b860f24f84efb4760fbf2",
		"4c0e071832d527694adea57b50dd7b2164c2a47c02940dcf26fa07c44d6d222a",
		"8d85f8467240628a94819b26bee26e3a9b2804334c63482deacec8d64ab4e1e7",
		"0b5000b73a53f0916c93c68f4b9b6ba8af5a10978634ae4f2237e1f3fbe324fa",
		"e66c57014a6156061ae669809ec5d735e484e8fcfd540e110c9b04f84c0b4504",
		"998e907bfbb34f71c66b6dc6c40fe98ca6d2d5a29755bc5a04824c36082a61d1",
		"5bc67471c189d78c76461dcab6141a733bdab3799d1d69e0c419119c92e82b3d",
		"1b8d0103e3a8d9ce8bda3bff71225be4b5bb18830466ae94f517321b7ecc6f94",
		"7a42e3892368f826928202014a6ca95a3d8d846df25088da80018663edf96b1c",
		"aed2b8245fdc8acc45eda51abc7d07e612c25f05cadd1579f3474f0bf1f6bdc6",
		"561f627b4213258dc8863498bb9b07c904c3c65a78c1a36bca329154d1ded213",
		"1209fe3bc3497e47376dfbd9df0600a17c63384c85f859671956d8289e5a0be8",
		"1664a6e0ea12d234b4911d011800bb0f8c1101a0f9a49a91ee6e2493e34d8e7b",
		"707d56f1f282aee234577e650bea2e7b18bb6131a499582be18876aba99d4b60",
		"4d75f61869104baa4ccff5be73311be9bdd6cc31779301dfc699479403c8a786",
		"0764c726a72f8e1d245f332a1d022fffdada0c4cb2a016886e4b33b66cb9a53f",
		"e9a5f5201eb3c3c856e0a224527af5ac7eb1767fb1aff9bd53ba41a60cde9785",
	}

	KAT39Nodes = []string{
		"af5570f5a1810b7af78caf4bc70a660f0df51e42baf91d4de5b2328de0e83dfc",
		"cd2662154e6d76b2b2b92e70c0cac3ccf534f9b74eb5b89819ec509083d00a50",
		"ad104051c516812ea5874ca3ff06d0258303623d04307c41ec80a7a18b332ef8",
		"d5688a52d55a02ec4aea5ec1eadfffe1c9e0ee6a4ddbe2377f98326d42dfc975",
		"8005f02d43fa06e7d0585fb64c961d57e318b27a145c857bcd3a6bdb413ff7fc",
		"9a18d3bc0a7d505ef45f985992270914cc02b44c91ccabba448c546a4b70f0f0",
		"827f3213c1de0d4c6277caccc1eeca325e45dfe2c65adce1943774218db61f88",
		"a3eb8db89fc5123ccfd49585059f292bc40a1c0d550b860f24f84efb4760fbf2",
		"4c0e071832d527694adea57b50dd7b2164c2a47c02940dcf26fa07c44d6d222a",
		"b8faf5f748f149b04018491a51334499fd8b6060c42a835f361fa9665562d12d",
		"8d85f8467240628a94819b26bee26e3a9b2804334c63482deacec8d64ab4e1e7",
		"0b5000b73a53f0916c93c68f4b9b6ba8af5a10978634ae4f2237e1f3fbe324fa",
		"6f3360ad3e99ab4ba39f2cbaf13da56ead8c9e697b03b901532ced50f7030fea",
		"508326f17c5f2769338cb00105faba3bf7862ca1e5c9f63ba2287e1f3cf2807a",
		"78b2b4162eb2c58b229288bbcb5b7d97c7a1154eed3161905fb0f180eba6f112",
		"e66c57014a6156061ae669809ec5d735e484e8fcfd540e110c9b04f84c0b4504",
		"998e907bfbb34f71c66b6dc6c40fe98ca6d2d5a29755bc5a04824c36082a61d1",
		"f4a0db79de0fee128fbe95ecf3509646203909dc447ae911aa29416bf6fcba21",
		"5bc67471c189d78c76461dcab6141a733bdab3799d1d69e0c419119c92e82b3d",
		"1b8d0103e3a8d9ce8bda3bff71225be4b5bb18830466ae94f517321b7ecc6f94",
		"0a4d7e66c92de549b765d9e2191027ff2a4ea8a7bd3eb04b0ed8ee063bad1f70",
		"61b3ff808934301578c9ed7402e3dd7dfe98b630acdf26d1fd2698a3c4a22710",
		"7a42e3892368f826928202014a6ca95a3d8d846df25088da80018663edf96b1c",
		"aed2b8245fdc8acc45eda51abc7d07e612c25f05cadd1579f3474f0bf1f6bdc6",
		"dd7efba5f1824103f1fa820a5c9e6cd90a82cf123d88bd035c7e5da0aba8a9ae",
		"561f627b4213258dc8863498bb9b07c904c3c65a78c1a36bca329154d1ded213",
		"1209fe3bc3497e47376dfbd9df0600a17c63384c85f859671956d8289e5a0be8",
		"6b4a3bd095c63d1dffae1ac03eb8264fdce7d51d2ac26ad0ebf9847f5b9be230",
		"4459f4d6c764dbaa6ebad24b0a3df644d84c3527c961c64aab2e39c58e027eb1",
		"77651b3eec6774e62545ae04900c39a32841e2b4bac80e2ba93755115252aae1",
		"d4fb5649422ff2eaf7b1c0b851585a8cfd14fb08ce11addb30075a96309582a7",
		"1664a6e0ea12d234b4911d011800bb0f8c1101a0f9a49a91ee6e2493e34d8e7b",
		"707d56f1f282aee234577e650bea2e7b18bb6131a499582be18876aba99d4b60",
		"0c9f36783b5929d43c97fe4b170d12137e6950ef1b3a8bd254b15bbacbfdee7f",
		"4d75f61869104baa4ccff5be73311be9bdd6cc31779301dfc699479403c8a786",
		"0764c726a72f8e1d245f332a1d022fffdada0c4cb2a016886e4b33b66cb9a53f",
		"c861552e9e17c41447d375c37928f9fa5d387d1e8470678107781c20a97ebc8f",
		"6a169105dcc487dbbae5747a0fd9b1d33a40320cf91cf9a323579139e7ff72aa",
		"e9a5f5201eb3c3c856e0a224527af5ac7eb1767fb1aff9bd53ba41a60cde9785",
	}
)

func hexHashList(hashes [][]byte) []string {
	var hexes []string
	for _, b := range hashes {
		hexes = append(hexes, hex.EncodeToString(b))
	}
	return hexes
}

func mustHex2Hash(t *testing.T, hexEncodedHash string) []byte {
	b, err := hex.DecodeString(hexEncodedHash)
	require.NoError(t, err)
	return b
}

type testDBLinear struct {
	nodes [][]byte
}

func (db *testDBLinear) Get(i uint64) ([]byte, error) {
	if int(i) < len(db.nodes) {
		return db.nodes[i], nil
	}
	return nil, fmt.Errorf("index %d out of range", i)
}

// Append adds a new node to the db and returns the index of the next addition
func (db *testDBLinear) Append(b []byte) (uint64, error) {
	db.nodes = append(db.nodes, b)
	return uint64(len(db.nodes)), nil
}

// TestDraftAddHashedLeaf tests that AddHashedLeaf creates the expected KAT39 MMR
func TestDraftAddHashedLeaf(t *testing.T) {
	db := &testDBLinear{}
	for e, leaf := range KAT39Leaves {
		leafHash := mustHex2Hash(t, leaf)
		iNext, err := AddHashedLeaf(db, sha256.New(), leafHash)
		assert.NoError(t, err)
		assert.Equal(t, MMRIndex(uint64(e+1)), iNext)
	}
	assert.Equal(t, len(KAT39Nodes), len(db.nodes))
	for i := 0; i < len(KAT39Nodes); i++ {
		assert.Equal(t, mustHex2Hash(t, KAT39Nodes[i]), db.nodes[i])
	}
}

// TestDraftAddLeafAccumulators tests that the AddHashedLeaf produces the expected accumulator states
func TestDraftAddLeafAccumulators(t *testing.T) {
	db := &testDBLinear{}
	for _, leaf := range KAT39Leaves {
		leafHash := mustHex2Hash(t, leaf)
		_, err := AddHashedLeaf(db, sha256.New(), leafHash)
		assert.NoError(t, err)
	}

	// Check the peaks are all in the expected places.
	for i, wantPeaks := range KAT39PeakHashes {
		peaks, err := PeakHashes(db, i)
		assert.NoError(t, err)
		assert.Equal(t, wantPeaks, hexHashList(peaks))
	}
}

// TestDraftKAT39PeakHashes tests that the peak indices match the KAT39 values
func TestDraftKAT39Peaks(t *testing.T) {
	for mmrIndex, wantPeaks := range KAT39PeakIndices {
		t.Run(fmt.Sprintf("%d", mmrIndex), func(t *testing.T) {
			if got := Peaks(mmrIndex); !reflect.DeepEqual(got, wantPeaks) {
				t.Errorf("Peaks() = %v, want %v", got, wantPeaks)
			}
		})
	}
}

// TestDraftKAT39PeakHashes tests that the peak indices obtain the expected KAT39 hashes
func TestDraftKAT39PeakHashes(t *testing.T) {

	db := NewCanonicalTestDB(t)

	for mmrIndex, wantPeaksHex := range KAT39PeakHashes {
		t.Run(fmt.Sprintf("%d", mmrIndex), func(t *testing.T) {
			peakHashes, err := PeakHashes(db, mmrIndex)
			require.NoError(t, err)
			peakHashesHex := hexHashList(peakHashes)
			if !reflect.DeepEqual(peakHashesHex, wantPeaksHex) {
				t.Errorf("PeakHashes() = %v, want %v", peakHashesHex, wantPeaksHex)
			}
		})
	}
}
