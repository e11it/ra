package validate

// Severity marks issue severity in validation report.
type Severity uint8

const (
	SeverityError Severity = iota
	SeverityWarning
)

// Control controls checker pipeline progression.
type Control uint8

const (
	Continue Control = iota
	StopRecord
	AbortAll
)

// Issue describes one validation problem.
type Issue struct {
	RecordIndex int
	Path        string
	Code        string
	Message     string
	Severity    Severity
}
