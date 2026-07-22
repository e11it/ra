package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/e11it/ra/pkg/validate"
)

func TestIsTombstoneCheck(t *testing.T) {
	ch, err := newIsTombstoneCheck(validate.Config{})
	require.NoError(t, err)

	cases := []struct {
		name     string
		key      string
		value    string
		expected validate.Control
		wantCode string
	}{
		{name: "json null with key", key: `"entity-1"`, value: "null", expected: validate.StopRecord},
		{name: "missing value", key: `"entity-1"`, value: "", expected: validate.Continue},
		{name: "null with missing key", value: "null", expected: validate.StopRecord, wantCode: "invalid_tombstone_key"},
		{name: "null with numeric key", key: "42", value: "null", expected: validate.StopRecord, wantCode: "invalid_tombstone_key"},
		{name: "null with empty key", key: `""`, value: "null", expected: validate.StopRecord, wantCode: "invalid_tombstone_key"},
		{name: "object value", value: "{\"x\":1}", expected: validate.Continue},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rep := validate.NewReport()
			ctx := &validate.CheckContext{
				Index:  2,
				Values: validate.NewValueStore([]byte(tc.value)),
			}
			ctrl := ch.Check(ctx, &validate.Record{Key: []byte(tc.key)}, rep)
			assert.Equal(t, tc.expected, ctrl)
			if tc.wantCode == "" {
				assert.False(t, rep.HasErrors())
				return
			}
			require.Len(t, rep.Issues(), 1)
			assert.Equal(t, tc.wantCode, rep.Issues()[0].Code)
			assert.Equal(t, "records[2].key", rep.Issues()[0].Path)
		})
	}
}
