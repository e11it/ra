//go:build !company

package ra

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/e11it/ra/pkg/validate"
)

func TestCreateBodyValidator_DefaultBuildIgnoresBodyValidationConfig(t *testing.T) {
	v, err := createBodyValidator(validate.Config{
		Enabled: true,
		Checks:  []string{"no_partition", "is_tombstone", "envelope"},
		StringLists: map[string][]string{
			"allowed_operations": {"CREATE"},
		},
	})
	require.NoError(t, err)
	assert.Nil(t, v)
}
