//go:build company

package company

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/e11it/ra/pkg/payloadvalidate"
	"github.com/e11it/ra/pkg/validate"
	_ "github.com/e11it/ra/pkg/validate/common"
)

func companyConfig() validate.Config {
	return companyConfigWithExtended(false)
}

func companyConfigWithExtended(extended bool) validate.Config {
	return validate.Config{
		Enabled:           true,
		ExtendedAvroTypes: extended,
		Checks: []string{
			noPartitionCheckName,
			"is_tombstone",
			envelopeCheckName,
			payloadCheckName,
			entityKeyCheckName,
		},
		StringLists: map[string][]string{
			"allowed_operations": defaultAllowedOperations,
		},
	}
}

func TestCompanyChecks_ValidBatch(t *testing.T) {
	v, err := payloadvalidate.NewValidatorFromConfig(companyConfig())
	require.NoError(t, err)
	require.NotNil(t, v)

	body := []byte(`{
  "records": [
    {
      "key": "k",
      "value": {
        "envelope": {
          "meta": {
            "eventId": "0192f8b9-7a1c-7c4e-9d3a-1f2c3a4b5c6d",
            "entityKey": "k",
            "operation": "UPDATE",
            "eventTime": 1745001234567,
            "eventTimeZone": "UTC"
          },
          "tech": {
            "sourceSystem": "crm",
            "schemaVersion": 0,
            "producedAt": 1745001234600
          }
        },
        "payload": {"x": 1},
        "payloadBefore": null
      }
    }
  ]
}`)
	assert.False(t, v.Validate(body).HasErrors())
}

func TestCompanyChecks_CollectsMultipleIssues(t *testing.T) {
	v, err := payloadvalidate.NewValidatorFromConfig(companyConfig())
	require.NoError(t, err)

	body := []byte(`{
  "records": [
    {
      "partition": 1,
      "key": "A",
      "value": {
        "envelope": {
          "meta": {
            "eventId": "bad",
            "entityKey": "B",
            "operation": 42,
            "eventTime": "x",
            "eventTimeZone": "+03:00"
          },
          "tech": {
            "sourceSystem": 1,
            "schemaVersion": 2,
            "producedAt": "z"
          }
        },
        "payload": null,
        "payloadBefore": null
      }
    }
  ]
}`)
	rep := v.Validate(body)
	require.True(t, rep.HasErrors())
	assert.Contains(t, rep.FormatList(), "partition_forbidden")
	assert.Contains(t, rep.FormatList(), "invalid_type")
	assert.Contains(t, rep.FormatList(), "invalid_payload_state")
}

func TestCompanyChecks_SchemaVersionValidation(t *testing.T) {
	tests := []struct {
		name           string
		schemaVersion  string
		expectedIssues []string
	}{
		{
			name:           "valid_zero",
			schemaVersion:  `"schemaVersion": 0,`,
			expectedIssues: nil,
		},
		{
			name:          "non_zero_int",
			schemaVersion: `"schemaVersion": 1,`,
			expectedIssues: []string{
				"records[0].envelope.tech.schemaVersion",
				"invalid_value",
			},
		},
		{
			name:          "string_type",
			schemaVersion: `"schemaVersion": "1.0.0",`,
			expectedIssues: []string{
				"records[0].envelope.tech.schemaVersion",
				"invalid_type",
			},
		},
		{
			name:          "missing_field",
			schemaVersion: ``,
			expectedIssues: []string{
				"records[0].envelope.tech.schemaVersion",
				"missing_field",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := payloadvalidate.NewValidatorFromConfig(companyConfig())
			require.NoError(t, err)

			body := []byte(`{
  "records": [
    {
      "key": "k",
      "value": {
        "envelope": {
          "meta": {
            "eventId": "0192f8b9-7a1c-7c4e-9d3a-1f2c3a4b5c6d",
            "entityKey": "k",
            "operation": "UPDATE",
            "eventTime": 1745001234567,
            "eventTimeZone": "UTC"
          },
          "tech": {
            "sourceSystem": "crm",
            ` + tt.schemaVersion + `
            "producedAt": 1745001234600
          }
        },
        "payload": {"x": 1},
        "payloadBefore": null
      }
    }
  ]
}`)

			rep := v.Validate(body)
			if len(tt.expectedIssues) == 0 {
				assert.False(t, rep.HasErrors())
				return
			}

			require.True(t, rep.HasErrors())
			issues := rep.FormatList()
			for _, issue := range tt.expectedIssues {
				assert.Contains(t, issues, issue)
			}
		})
	}
}

func TestCompanyChecks_UnknownCheckFails(t *testing.T) {
	_, err := payloadvalidate.NewValidatorFromConfig(validate.Config{
		Enabled: true,
		Checks:  []string{"missing_company_check"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown check")
}

func TestCompanyChecks_TombstoneWithPartitionFails(t *testing.T) {
	v, err := payloadvalidate.NewValidatorFromConfig(companyConfig())
	require.NoError(t, err)
	body := []byte(`{"records":[{"key":"k","partition":0,"value":null}]}`)
	rep := v.Validate(body)
	require.True(t, rep.HasErrors())
	assert.Contains(t, rep.FormatList(), "partition_forbidden")
}

func TestCompanyChecks_TombstoneRequiresExplicitNullAndStringKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		record   string
		wantCode string
	}{
		{name: "valid tombstone", record: `{"key":"k","value":null}`},
		{name: "missing value", record: `{"key":"k"}`, wantCode: "missing_value"},
		{name: "missing key", record: `{"value":null}`, wantCode: "invalid_tombstone_key"},
		{name: "numeric key", record: `{"key":42,"value":null}`, wantCode: "invalid_tombstone_key"},
		{name: "empty key", record: `{"key":"","value":null}`, wantCode: "invalid_tombstone_key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v, err := payloadvalidate.NewValidatorFromConfig(companyConfig())
			require.NoError(t, err)
			rep := v.Validate([]byte(`{"records":[` + tt.record + `]}`))
			if tt.wantCode == "" {
				assert.False(t, rep.HasErrors())
				return
			}
			require.True(t, rep.HasErrors())
			assert.Contains(t, rep.FormatList(), tt.wantCode)
		})
	}
}

func TestCompanyChecks_ExtendedAvroTypes_ValidBatch(t *testing.T) {
	v, err := payloadvalidate.NewValidatorFromConfig(companyConfigWithExtended(true))
	require.NoError(t, err)
	require.NotNil(t, v)

	body := []byte(`{
  "records": [
    {
      "key": "k",
      "value": {
        "envelope": {
          "meta": {
            "eventId": "0192f8b9-7a1c-7c4e-9d3a-1f2c3a4b5c6d",
            "entityKey": "k",
            "operation": "UPDATE",
            "eventTime": "2024-04-19T10:00:00Z",
            "eventTimeZone": "UTC",
            "businessDate": {"date": "2024-01-15"}
          },
          "tech": {
            "sourceSystem": "crm",
            "schemaVersion": 0,
            "producedAt": "2024-04-19T10:00:00.001Z",
            "traceId": {"string": "0192f8b9-7a1c-7c4e-9d3a-1f2c3a4b5c6d"}
          }
        },
        "payload": {"com.example.Payload": {"x": 1}},
        "payloadBefore": null
      }
    }
  ]
}`)
	assert.False(t, v.Validate(body).HasErrors())
}

func TestCompanyChecks_ExtendedAvroTypes_TraceIDMustBeUUID(t *testing.T) {
	v, err := payloadvalidate.NewValidatorFromConfig(companyConfigWithExtended(true))
	require.NoError(t, err)
	require.NotNil(t, v)

	body := []byte(`{
  "records": [
    {
      "key": "k",
      "value": {
        "envelope": {
          "meta": {
            "eventId": "0192f8b9-7a1c-7c4e-9d3a-1f2c3a4b5c6d",
            "entityKey": "k",
            "operation": "UPDATE",
            "eventTime": "2024-04-19T10:00:00Z",
            "eventTimeZone": "UTC"
          },
          "tech": {
            "sourceSystem": "crm",
            "schemaVersion": 0,
            "producedAt": "2024-04-19T10:00:00.001Z",
            "traceId": {"string": "not-a-uuid"}
          }
        },
        "payload": {"com.example.Payload": {"x": 1}},
        "payloadBefore": null
      }
    }
  ]
}`)

	rep := v.Validate(body)
	require.True(t, rep.HasErrors())
	assert.Contains(t, rep.FormatList(), "records[0].envelope.tech.traceId")
	assert.Contains(t, rep.FormatList(), "invalid_format")
}

func TestCompanyChecks_ExtendedAvroTypes_EntityKeyMustBePlainString(t *testing.T) {
	v, err := payloadvalidate.NewValidatorFromConfig(companyConfigWithExtended(true))
	require.NoError(t, err)
	require.NotNil(t, v)

	body := []byte(`{
  "records": [
    {
      "key": "k",
      "value": {
        "envelope": {
          "meta": {
            "eventId": "0192f8b9-7a1c-7c4e-9d3a-1f2c3a4b5c6d",
            "entityKey": {"string": "k"},
            "operation": "UPDATE",
            "eventTime": "2024-04-19T10:00:00Z",
            "eventTimeZone": "UTC"
          },
          "tech": {
            "sourceSystem": "crm",
            "schemaVersion": 0,
            "producedAt": "2024-04-19T10:00:00.001Z"
          }
        },
        "payload": {"com.example.Payload": {"x": 1}},
        "payloadBefore": null
      }
    }
  ]
}`)

	rep := v.Validate(body)
	require.True(t, rep.HasErrors())
	assert.Contains(t, rep.FormatList(), "records[0].envelope.meta.entityKey")
	assert.Contains(t, rep.FormatList(), "invalid_type")
}

func TestCompanyChecks_ExtendedAvroTypes_StringsRejectedWhenDisabled(t *testing.T) {
	v, err := payloadvalidate.NewValidatorFromConfig(companyConfigWithExtended(false))
	require.NoError(t, err)
	require.NotNil(t, v)

	body := []byte(`{
  "records": [
    {
      "key": "k",
      "value": {
        "envelope": {
          "meta": {
            "eventId": "0192f8b9-7a1c-7c4e-9d3a-1f2c3a4b5c6d",
            "entityKey": "k",
            "operation": "UPDATE",
            "eventTime": "2024-04-19T10:00:00Z",
            "eventTimeZone": "UTC"
          },
          "tech": {
            "sourceSystem": "crm",
            "schemaVersion": 0,
            "producedAt": "2024-04-19T10:00:00.001Z"
          }
        },
        "payload": {"x": 1},
        "payloadBefore": null
      }
    }
  ]
}`)

	rep := v.Validate(body)
	require.True(t, rep.HasErrors())
	assert.Contains(t, rep.FormatList(), "records[0].envelope.meta.eventTime")
	assert.Contains(t, rep.FormatList(), "records[0].envelope.tech.producedAt")
}
