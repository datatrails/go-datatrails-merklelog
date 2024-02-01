package mmrblobs

import "errors"

var (
	ErrNotleaf = errors.New("mmr node not a leaf")
)
