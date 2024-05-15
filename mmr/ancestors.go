package mmr

func LeftAncestors(i uint64) []uint64 {

	height := IndexHeight(i)
	if height < 1 {
		return nil
	}
	height -= 1

	var ancestors []uint64

	for IndexHeight(i) > height {
		iLeft := i - (2 << height)
		ancestors = append(ancestors, iLeft)
		i += 1
		height += 1
	}
	return ancestors
}

func Ancestors(i uint64) []uint64 {

	height := IndexHeight(i)
	if height < 1 {
		return nil
	}
	height -= 1

	var ancestors []uint64

	for IndexHeight(i) > height {
		iLeft := i - (2 << height)
		iRight := iLeft + SiblingOffset(height)
		ancestors = append(ancestors, iLeft)
		ancestors = append(ancestors, iRight)
		i += 1
		height += 1
	}
	return ancestors
}
