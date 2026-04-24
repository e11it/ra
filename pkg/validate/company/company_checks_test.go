//go:build company

package company

import (
	"testing"

	"github.com/e11it/ra/pkg/payloadvalidate"
	"github.com/e11it/ra/pkg/validate"
	_ "github.com/e11it/ra/pkg/validate/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func companyConfig() validate.Config {
	return validate.Config{
		Enabled: true,
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

	body := []byte(`{"records":[{"key":"k","value":{"envelope":{"meta":{"eventId":"0192f8b9-7a1c-7c4e-9d3a-1f2c3a4b5c6d","entityKey":"k","operation":"UPDATE","eventTime":1745001234567890,"eventTimeZone":"UTC"},"tech":{"sourceSystem":"crm","schemaVersion":"1.0.0","producedAt":1745001234600000}},"payload":{"x":1},"payloadBefore":null}}]}`)
	assert.False(t, v.Validate(body).HasErrors())
}

func TestCompanyChecks_CollectsMultipleIssues(t *testing.T) {
	v, err := payloadvalidate.NewValidatorFromConfig(companyConfig())
	require.NoError(t, err)

	body := []byte(`{"records":[{"partition":1,"key":"A","value":{"envelope":{"meta":{"eventId":"bad","entityKey":"B","operation":42,"eventTime":"x","eventTimeZone":"+03:00"},"tech":{"sourceSystem":1,"schemaVersion":2,"producedAt":"z"}},"payload":null,"payloadBefore":null}}]}`)
	rep := v.Validate(body)
	require.True(t, rep.HasErrors())
	assert.Contains(t, rep.FormatList(), "partition_forbidden")
	assert.Contains(t, rep.FormatList(), "invalid_type")
	assert.Contains(t, rep.FormatList(), "invalid_payload_state")
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
