package validate

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCheck struct {
	name string
}

func (c testCheck) Name() string { return c.name }

func (c testCheck) Check(_ *CheckContext, _ *Record, _ *Report) Control { return Continue }

func TestRegistry_BuildUnknownCheck(t *testing.T) {
	r := NewRegistry()
	_, err := r.Build(Config{Enabled: true, Checks: []string{"missing"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown check")
}

func TestRegistry_BuildPreservesOrder(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Register("a", func(_ Config) (RecordChecker, error) { return testCheck{name: "a"}, nil }))
	require.NoError(t, r.Register("b", func(_ Config) (RecordChecker, error) { return testCheck{name: "b"}, nil }))

	checks, err := r.Build(Config{Enabled: true, Checks: []string{"b", "a"}})
	require.NoError(t, err)
	require.Len(t, checks, 2)
	assert.Equal(t, "b", checks[0].Name())
	assert.Equal(t, "a", checks[1].Name())
}

func TestRegistry_FactoryErrorWrapped(t *testing.T) {
	sentinel := errors.New("boom")
	r := NewRegistry()
	require.NoError(t, r.Register("x", func(_ Config) (RecordChecker, error) { return nil, sentinel }))
	_, err := r.Build(Config{Enabled: true, Checks: []string{"x"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "build check")
	assert.ErrorIs(t, err, sentinel)
}
