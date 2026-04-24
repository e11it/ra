package validate

// BodyValidator validates Kafka REST v2 produce request bodies.
type BodyValidator interface {
	Validate(body []byte) *Report
}

// RecordChecker validates one record from records[].
type RecordChecker interface {
	// Name is a stable checker identifier used in configuration.
	Name() string
	// Check validates one record and appends issues into report.
	Check(ctx *CheckContext, rec *Record, rep *Report) Control
}

// CheckerFactory builds a checker instance from validator config.
type CheckerFactory func(cfg Config) (RecordChecker, error)
