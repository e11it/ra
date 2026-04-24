package validate

import (
	"fmt"
	"strings"
)

// Report accumulates validation issues.
type Report struct {
	issues []Issue
}

// NewReport allocates an empty report.
func NewReport() *Report {
	return &Report{issues: make([]Issue, 0, 4)}
}

// Add appends a raw issue.
func (r *Report) Add(issue Issue) {
	if r == nil {
		return
	}
	r.issues = append(r.issues, issue)
}

// AddError appends error severity issue.
func (r *Report) AddError(rec int, path, code, msg string) {
	r.Add(Issue{
		RecordIndex: rec,
		Path:        path,
		Code:        code,
		Message:     msg,
		Severity:    SeverityError,
	})
}

// AddWarning appends warning severity issue.
func (r *Report) AddWarning(rec int, path, code, msg string) {
	r.Add(Issue{
		RecordIndex: rec,
		Path:        path,
		Code:        code,
		Message:     msg,
		Severity:    SeverityWarning,
	})
}

// Issues returns a copy of all report issues.
func (r *Report) Issues() []Issue {
	if r == nil || len(r.issues) == 0 {
		return nil
	}
	out := make([]Issue, len(r.issues))
	copy(out, r.issues)
	return out
}

// HasErrors reports whether report has at least one error.
func (r *Report) HasErrors() bool {
	if r == nil {
		return false
	}
	for _, issue := range r.issues {
		if issue.Severity == SeverityError {
			return true
		}
	}
	return false
}

// SummaryLine returns short first-error summary safe for headers.
func (r *Report) SummaryLine() string {
	if r == nil || len(r.issues) == 0 {
		return ""
	}
	firstIdx := -1
	errorsCount := 0
	for i, issue := range r.issues {
		if issue.Severity == SeverityError {
			errorsCount++
			if firstIdx < 0 {
				firstIdx = i
			}
		}
	}
	if firstIdx < 0 {
		return ""
	}
	base := formatIssue(r.issues[firstIdx])
	if errorsCount == 1 {
		return base
	}
	return fmt.Sprintf("%s (+%d more)", base, errorsCount-1)
}

// FormatList returns multiline list of all error issues.
func (r *Report) FormatList() string {
	if r == nil || len(r.issues) == 0 {
		return ""
	}
	lines := make([]string, 0, len(r.issues))
	for _, issue := range r.issues {
		if issue.Severity != SeverityError {
			continue
		}
		lines = append(lines, formatIssue(issue))
	}
	return strings.Join(lines, "\n")
}

// AsError returns nil when report has no errors.
func (r *Report) AsError() error {
	if !r.HasErrors() {
		return nil
	}
	return reportError{report: r}
}

type reportError struct {
	report *Report
}

func (e reportError) Error() string {
	return e.report.FormatList()
}

func formatIssue(issue Issue) string {
	recordPrefix := "request"
	if issue.RecordIndex >= 0 {
		recordPrefix = fmt.Sprintf("records[%d]", issue.RecordIndex)
	}
	path := strings.TrimSpace(issue.Path)
	code := strings.TrimSpace(issue.Code)
	msg := strings.TrimSpace(issue.Message)
	switch {
	case path != "" && code != "":
		return fmt.Sprintf("%s %s: %s: %s", recordPrefix, path, code, msg)
	case code != "":
		return fmt.Sprintf("%s: %s: %s", recordPrefix, code, msg)
	case path != "":
		return fmt.Sprintf("%s %s: %s", recordPrefix, path, msg)
	default:
		return fmt.Sprintf("%s: %s", recordPrefix, msg)
	}
}
