package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReport_AsError(t *testing.T) {
	rep := NewReport()
	require.Nil(t, rep.AsError())

	rep.AddWarning(0, "records[0].x", "warn", "test warning")
	require.Nil(t, rep.AsError())

	rep.AddError(1, "records[1].y", "bad", "test error")
	require.Error(t, rep.AsError())
	assert.True(t, rep.HasErrors())
	assert.Contains(t, rep.SummaryLine(), "bad")
	assert.Contains(t, rep.FormatList(), "records[1].y")
}
