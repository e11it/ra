package vcheck

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/e11it/ra/pkg/validate"
)

func TestAsString(t *testing.T) {
	rep := validate.NewReport()
	_, ok := AsString(rep, 0, "records[0].x", 42)
	assert.False(t, ok)
	assert.True(t, rep.HasErrors())
}

func TestUnionHelpers(t *testing.T) {
	rep := validate.NewReport()
	_, isNull, ok := UnionString(rep, 0, "records[0].x", map[string]any{"string": "v"})
	assert.True(t, ok)
	assert.False(t, isNull)
	assert.False(t, rep.HasErrors())
}

func TestIsSemverLike(t *testing.T) {
	rep := validate.NewReport()
	assert.True(t, IsSemverLike(rep, 0, "records[0].schemaVersion", "1.2.3"))
	assert.False(t, IsSemverLike(rep, 0, "records[0].schemaVersion", "abc"))
	assert.True(t, rep.HasErrors())
}

func TestAsTimestampMicros(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		extended bool
		ok       bool
	}{
		{
			name:     "integer value accepted in standard mode",
			value:    float64(1745001234567890),
			extended: false,
			ok:       true,
		},
		{
			name:     "integer value accepted in extended mode",
			value:    float64(1745001234567890),
			extended: true,
			ok:       true,
		},
		{
			name:     "rfc3339 string accepted in extended mode",
			value:    "2024-04-19T10:00:00.123456Z",
			extended: true,
			ok:       true,
		},
		{
			name:     "rfc3339 string rejected in standard mode",
			value:    "2024-04-19T10:00:00.123456Z",
			extended: false,
			ok:       false,
		},
		{
			name:     "invalid string rejected in extended mode",
			value:    "not-a-timestamp",
			extended: true,
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rep := validate.NewReport()
			got, ok := AsTimestampMicros(rep, 0, "records[0].envelope.meta.eventTime", tt.value, tt.extended)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.False(t, rep.HasErrors())
				assert.NotZero(t, got)
				return
			}
			assert.True(t, rep.HasErrors())
		})
	}
}

func TestAsTimestampMicros_RFC3339Value(t *testing.T) {
	rep := validate.NewReport()
	got, ok := AsTimestampMicros(rep, 0, "records[0].envelope.meta.eventTime", "2024-04-19T10:00:00Z", true)
	require.True(t, ok)
	assert.Equal(t, int64(1713520800000000), got)
}
