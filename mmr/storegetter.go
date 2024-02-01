package mmr

type indexStoreGetter interface {
	Get(i uint64) ([]byte, error)
}
