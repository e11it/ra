package validate

// Config describes runtime body-validation configuration.
type Config struct {
	// Enabled toggles body validation.
	Enabled bool

	// Checks defines ordered checker names.
	Checks []string

	// StringLists stores checker-specific list options.
	StringLists map[string][]string
}

// IsEmpty reports whether validation is effectively disabled.
func (c Config) IsEmpty() bool {
	return !c.Enabled || len(c.Checks) == 0
}

// List returns a copy of a configured list option.
func (c Config) List(name string) []string {
	if c.StringLists == nil {
		return nil
	}
	src := c.StringLists[name]
	if len(src) == 0 {
		return nil
	}
	dst := make([]string, 0, len(src))
	for _, item := range src {
		if item != "" {
			dst = append(dst, item)
		}
	}
	return dst
}
