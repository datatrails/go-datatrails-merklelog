package massifs

import "github.com/datatrails/go-datatrails-merklelog/mmr"

// Methods for working with the mmrblob peak stack

// PeakStackMap builds a map from mmr indices to peak stack entries
// massifHeight is the 1 based height (not the height index)
func PeakStackMap(massifHeight uint8, mmrIndex uint64) map[uint64]int {

	if massifHeight == 0 {
		return nil
	}

	// XXX:TODO there is likely a more efficient way to do this using
	// PeaksBitmap or a variation of it, but this isn't a terribly hot path.
	stackMap := map[uint64]int{}
	iPeaks := mmr.Peaks(mmrIndex)
	for i, ip := range iPeaks {
		if mmr.IndexHeight(ip) < uint64(massifHeight-1) {
			continue
		}
		stackMap[ip] = i
	}

	return stackMap
}
