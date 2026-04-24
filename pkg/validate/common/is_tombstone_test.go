package common

import (
	"testing"

	"github.com/e11it/ra/pkg/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsTombstoneCheck(t *testing.T) {
	ch, err := newIsTombstoneCheck(validate.Config{})
	require.NoError(t, err)
	rep := validate.NewReport()

	cases := []struct {
		name     string
		value    string
		expected validate.Control
	}{
		{name: "json_null", value: "null", expected: validate.StopRecord},
		{name: "empty_value", value: "", expected: validate.StopRecord},
		{name: "object_value", value: "{\"x\":1}", expected: validate.Continue},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &validate.CheckContext{
				Values: validate.NewValueStore([]byte(tc.value)),
			}
			ctrl := ch.Check(ctx, &validate.Record{}, rep)
			assert.Equal(t, tc.expected, ctrl)
		})
	}
	assert.False(t, rep.HasErrors())
}
