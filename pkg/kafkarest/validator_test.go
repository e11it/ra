package kafkarest

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultConfig — конфиг "включено всё из каталога".
func defaultConfig() Config {
	return Config{
		Enabled:           true,
		AllowedOperations: defaultAllowedOperations,
		Checks:            []string{entityKeyMatchCheckName, operationAllowedCheckName},
	}
}

func buildDefaultValidator(t *testing.T) BodyValidator {
	t.Helper()
	v, err := NewValidatorFromConfig(defaultConfig())
	require.NoError(t, err)
	require.NotNil(t, v)
	return v
}

func TestValidator_DisabledOrEmpty(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
	}{
		{"disabled", Config{Enabled: false, Checks: []string{entityKeyMatchCheckName}}},
		{"no_checks", Config{Enabled: true, Checks: nil}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, err := NewValidatorFromConfig(tc.cfg)
			require.NoError(t, err)
			assert.Nil(t, v, "выключенный/пустой конфиг должен давать nil BodyValidator")
		})
	}
}

func TestValidator_EmptyBody(t *testing.T) {
	v := buildDefaultValidator(t)
	assert.NoError(t, v.Validate(nil))
	assert.NoError(t, v.Validate([]byte("")))
	assert.NoError(t, v.Validate([]byte("   \n")))
}

func TestValidator_ValidBatch(t *testing.T) {
	body := []byte(`{
		"value_schema_id": 16,
		"records": [
			{"key": "554123", "value": {"envelope": {"meta": {"entityKey": "554123", "operation": "UPDATE"}}}},
			{"key": "evt-1", "value": {"envelope": {"meta": {"entityKey": "evt-1", "operation": "EVENT"}}}},
			{"key": "x", "value": {"envelope": {"meta": {"entityKey": "x", "operation": "CREATE"}}}}
		]
	}`)
	assert.NoError(t, buildDefaultValidator(t).Validate(body))
}

func TestValidator_Tombstone(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			name: "only_tombstone",
			body: `{"records": [{"key": "554123", "value": null}]}`,
		},
		{
			name: "mixed_with_envelope",
			body: `{"records": [
				{"key": "554123", "value": {"envelope": {"meta": {"entityKey": "554123", "operation": "DELETE"}}}},
				{"key": "554123", "value": null}
			]}`,
		},
		{
			name: "tombstone_with_partition",
			body: `{"records": [{"key": "k", "value": null, "partition": 1}]}`,
		},
	}
	v := buildDefaultValidator(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NoError(t, v.Validate([]byte(tc.body)))
		})
	}
}

func TestValidator_EntityKeyMismatch(t *testing.T) {
	body := []byte(`{"records": [
		{"key": "A", "value": {"envelope": {"meta": {"entityKey": "B", "operation": "UPDATE"}}}}
	]}`)
	err := buildDefaultValidator(t).Validate(body)
	require.Error(t, err)
	var verr *ValidationError
	require.True(t, errors.As(err, &verr))
	assert.Equal(t, 0, verr.Index)
	assert.Equal(t, entityKeyMatchCheckName, verr.Check)
}

func TestValidator_EventWithEmptyEntityKey(t *testing.T) {
	// EVENT + пустой entityKey + отсутствие key → валидно.
	body := []byte(`{"records": [
		{"value": {"envelope": {"meta": {"entityKey": "", "operation": "EVENT"}}}}
	]}`)
	assert.NoError(t, buildDefaultValidator(t).Validate(body))
}

func TestValidator_OperationNotAllowed(t *testing.T) {
	body := []byte(`{"records": [
		{"key": "k", "value": {"envelope": {"meta": {"entityKey": "k", "operation": "WEIRD"}}}}
	]}`)
	err := buildDefaultValidator(t).Validate(body)
	require.Error(t, err)
	var verr *ValidationError
	require.True(t, errors.As(err, &verr))
	assert.Equal(t, operationAllowedCheckName, verr.Check)
}

func TestValidator_MalformedJSON(t *testing.T) {
	err := buildDefaultValidator(t).Validate([]byte(`{not-json}`))
	require.Error(t, err)
	var verr *ValidationError
	require.True(t, errors.As(err, &verr))
	assert.Equal(t, -1, verr.Index)
}

func TestValidator_EmptyRecords(t *testing.T) {
	err := buildDefaultValidator(t).Validate([]byte(`{"records": []}`))
	require.Error(t, err)
	var verr *ValidationError
	require.True(t, errors.As(err, &verr))
	assert.Equal(t, -1, verr.Index)
}

func TestValidator_MissingEnvelope(t *testing.T) {
	body := []byte(`{"records": [
		{"key": "k", "value": {"payload": {"x": 1}}}
	]}`)
	err := buildDefaultValidator(t).Validate(body)
	require.Error(t, err)
	var verr *ValidationError
	require.True(t, errors.As(err, &verr))
}

func TestRegistry_UnknownCheck(t *testing.T) {
	_, err := NewValidatorFromConfig(Config{
		Enabled: true,
		Checks:  []string{"does_not_exist"},
	})
	require.Error(t, err)
}

func TestBuilder_CheckersPreserveOrder(t *testing.T) {
	// Если заказан только operation_allowed, entity_key_match не должен появиться.
	v, err := NewValidatorFromConfig(Config{
		Enabled: true,
		Checks:  []string{operationAllowedCheckName},
	})
	require.NoError(t, err)
	require.NotNil(t, v)

	body := []byte(`{"records": [
		{"key": "A", "value": {"envelope": {"meta": {"entityKey": "B", "operation": "UPDATE"}}}}
	]}`)
	assert.NoError(t, v.Validate(body),
		"entity_key_match не должен срабатывать, если его нет в Checks")
}
