package validate

// CheckContext describes one checker invocation.
type CheckContext struct {
	Index  int
	Values *ValueStore
}
