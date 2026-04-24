package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValueStore_IsNull(t *testing.T) {
	assert.True(t, NewValueStore(nil).IsNull())
	assert.True(t, NewValueStore([]byte("null")).IsNull())
	assert.True(t, NewValueStore([]byte("  ")).IsNull())
	assert.False(t, NewValueStore([]byte(`{"x":1}`)).IsNull())
}

func TestValueStore_RootAndCache(t *testing.T) {
	s := NewValueStore([]byte(`{"x":1}`))
	root, ok, err := s.Root()
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, float64(1), root["x"])

	called := 0
	v1, err := s.Cached("key", func() (any, error) {
		called++
		return "ok", nil
	})
	require.NoError(t, err)
	v2, err := s.Cached("key", func() (any, error) {
		called++
		return "no", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", v1)
	assert.Equal(t, "ok", v2)
	assert.Equal(t, 1, called)
}
