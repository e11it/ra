package vcheck

import (
	"testing"

	"github.com/e11it/ra/pkg/validate"
	"github.com/stretchr/testify/assert"
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
	_, isNull, ok = UnionInt(rep, 0, "records[0].y", nil)
	assert.True(t, ok)
	assert.True(t, isNull)
	assert.False(t, rep.HasErrors())
}

func TestIsSemverLike(t *testing.T) {
	rep := validate.NewReport()
	assert.True(t, IsSemverLike(rep, 0, "records[0].schemaVersion", "1.2.3"))
	assert.False(t, IsSemverLike(rep, 0, "records[0].schemaVersion", "abc"))
	assert.True(t, rep.HasErrors())
}
