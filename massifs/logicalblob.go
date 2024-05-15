package massifs

// LogicalBlob idenfies a logical blob like "first" or "last" or "head"
type LogicalBlob int

const (
	FirstBlob LogicalBlob = iota
	LastBlob
)
