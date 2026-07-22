//go:build !company

package ra

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/e11it/ra/pkg/validate"
)

func TestCreateBodyValidator_DefaultBuildRejectsEnabledBodyValidation(t *testing.T) {
	v, err := createBodyValidator(validate.Config{
		Enabled: true,
		Checks:  []string{"no_partition", "is_tombstone", "envelope"},
		StringLists: map[string][]string{
			"allowed_operations": {"CREATE"},
		},
	})
	require.ErrorContains(t, err, "body validation requires company build")
	assert.Nil(t, v)
}

func TestCreateBodyValidator_DefaultBuildAllowsDisabledBodyValidation(t *testing.T) {
	v, err := createBodyValidator(validate.Config{Enabled: false})
	require.NoError(t, err)
	assert.Nil(t, v)
}
