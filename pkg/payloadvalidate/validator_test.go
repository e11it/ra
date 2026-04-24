package payloadvalidate

import (
	"testing"

	"github.com/e11it/ra/pkg/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordSpyCheck struct {
	called int
}

func (c *recordSpyCheck) Name() string { return "record_spy" }

func (c *recordSpyCheck) Check(_ *validate.CheckContext, _ *validate.Record, _ *validate.Report) validate.Control {
	c.called++
	return validate.Continue
}

type stopOnSecondCheck struct {
	calls []int
}

func (c *stopOnSecondCheck) Name() string { return "stop_on_second" }

func (c *stopOnSecondCheck) Check(ctx *validate.CheckContext, _ *validate.Record, _ *validate.Report) validate.Control {
	c.calls = append(c.calls, ctx.Index)
	if ctx.Index == 1 {
		return validate.StopRecord
	}
	return validate.Continue
}

func buildValidator(t *testing.T, checks ...validate.RecordChecker) validate.BodyValidator {
	t.Helper()
	return NewValidator(checks)
}

func TestValidator_EmptyBody(t *testing.T) {
	v := buildValidator(t)
	rep := v.Validate(nil)
	require.NotNil(t, rep)
	assert.False(t, rep.HasErrors())
}

func TestValidator_CheckersAreCalled(t *testing.T) {
	spy := &recordSpyCheck{}
	v := buildValidator(t, spy)
	body := []byte(`{"records":[{"value":{"a":1}},{"value":{"b":2}}]}`)
	rep := v.Validate(body)
	assert.False(t, rep.HasErrors())
	assert.Equal(t, 2, spy.called)
}

func TestValidator_MalformedJSON(t *testing.T) {
	v := buildValidator(t, &recordSpyCheck{})
	rep := v.Validate([]byte(`{not-json}`))
	require.True(t, rep.HasErrors())
	assert.Contains(t, rep.SummaryLine(), "malformed_body")
}

func TestValidator_EmptyRecords(t *testing.T) {
	v := buildValidator(t, &recordSpyCheck{})
	rep := v.Validate([]byte(`{"records":[]}`))
	require.True(t, rep.HasErrors())
	assert.Contains(t, rep.SummaryLine(), "empty_records")
}

func TestValidator_ControlFlow(t *testing.T) {
	stop := &stopOnSecondCheck{}
	spy := &recordSpyCheck{}
	v := buildValidator(t, stop, spy)
	body := []byte(`{"records":[{"value":{"a":1}},{"value":{"b":2}},{"value":{"c":3}}]}`)
	rep := v.Validate(body)
	assert.False(t, rep.HasErrors())
	assert.Equal(t, []int{0, 1, 2}, stop.calls)
	assert.Equal(t, 2, spy.called)
}

func TestValidator_ParsesSchemaFields(t *testing.T) {
	spy := &recordSpyCheck{}
	v := buildValidator(t, spy)
	body := []byte(`{"value_schema_id":32,"records":[{"value":{"x":1}}]}`)
	rep := v.Validate(body)
	assert.False(t, rep.HasErrors())
	assert.Equal(t, 1, spy.called)
}
