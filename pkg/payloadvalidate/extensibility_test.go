package payloadvalidate

import (
	"testing"

	"github.com/e11it/ra/pkg/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type alwaysFailCheckExt struct{}

func (alwaysFailCheckExt) Name() string { return "always_fail" }

func (alwaysFailCheckExt) Check(ctx *validate.CheckContext, _ *validate.Record, rep *validate.Report) validate.Control {
	rep.AddError(ctx.Index, "records", "forced_failure", "forced failure")
	return validate.Continue
}

func TestValidator_ComposableCheckers(t *testing.T) {
	v := NewValidator([]validate.RecordChecker{alwaysFailCheckExt{}})

	body := []byte(`{"records":[{"value":{"x":1}}]}`)
	rep := v.Validate(body)
	require.NotNil(t, rep)
	require.True(t, rep.HasErrors())
	assert.Contains(t, rep.FormatList(), "forced_failure")
}
