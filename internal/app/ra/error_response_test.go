package ra

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatPayloadValidationMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		details RAErrorDetails
		want    string
	}{
		{
			name:    "base_only_no_trace_no_errors",
			details: RAErrorDetails{},
			want:    payloadValidationMessageBase,
		},
		{
			name: "duplicate_codes_and_trace",
			details: RAErrorDetails{
				TraceID: "99ff77f2-4cf1-46b1-a359-52b1f13d3092",
				Errors: []RAValidationErrorItem{
					{Code: "missing_field"},
					{Code: "missing_field"},
				},
			},
			want: "Ra: payload validation errors. Problems: [missing_field]. Trace ID: 99ff77f2-4cf1-46b1-a359-52b1f13d3092.",
		},
		{
			name: "unique_codes_preserve_first_seen_order",
			details: RAErrorDetails{
				TraceID: "t",
				Errors: []RAValidationErrorItem{
					{Code: "b"},
					{Code: "a"},
					{Code: "b"},
				},
			},
			want: "Ra: payload validation errors. Problems: [b, a]. Trace ID: t.",
		},
		{
			name: "empty_codes_trace_only",
			details: RAErrorDetails{
				TraceID: "abc",
				Errors: []RAValidationErrorItem{
					{Code: "   "},
					{Code: ""},
				},
			},
			want: "Ra: payload validation errors. Trace ID: abc.",
		},
		{
			name: "codes_without_trace",
			details: RAErrorDetails{
				Errors: []RAValidationErrorItem{{Code: "x"}},
			},
			want: "Ra: payload validation errors. Problems: [x]",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, FormatPayloadValidationMessage(tt.details))
		})
	}
}
