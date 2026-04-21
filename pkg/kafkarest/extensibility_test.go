package kafkarest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// alwaysFailCheck демонстрирует, что внешний чекер можно составлять в цепочку
// через NewValidator напрямую (не обращаясь к встроенному registry).
type alwaysFailCheck struct{}

func (alwaysFailCheck) Name() string { return "always_fail" }

func (alwaysFailCheck) Check(ctx CheckContext, _ *Record) error {
	if ctx.IsTombstone {
		return nil
	}
	return NewValidationError(ctx.Index, "always_fail", "forced failure")
}

func TestValidator_ComposableCheckers(t *testing.T) {
	v := NewValidator([]RecordChecker{&entityKeyMatchCheck{}, alwaysFailCheck{}})

	t.Run("fails_on_second_check", func(t *testing.T) {
		body := []byte(`{"records": [
			{"key": "k", "value": {"envelope": {"meta": {"entityKey": "k", "operation": "CREATE"}}}}
		]}`)
		err := v.Validate(body)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "always_fail")
	})

	t.Run("tombstone_is_skipped", func(t *testing.T) {
		body := []byte(`{"records": [{"key": "k", "value": null}]}`)
		assert.NoError(t, v.Validate(body))
	})

	t.Run("index_is_preserved", func(t *testing.T) {
		body := []byte(`{"records": [
			{"key": "k", "value": null},
			{"key": "k", "value": {"envelope": {"meta": {"entityKey": "k", "operation": "CREATE"}}}}
		]}`)
		err := v.Validate(body)
		require.Error(t, err)
		assert.Contains(t, err.Error(), fmt.Sprintf("records[%d]", 1))
	})
}
