package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSimpleAccessController_RejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		config    *Config
		wantError string
	}{
		{
			name:      "nil config",
			config:    nil,
			wantError: "auth config is nil",
		},
		{
			name:      "empty ACL",
			config:    &Config{},
			wantError: "auth acl must contain at least one rule",
		},
		{
			name: "invalid URL validation regexp",
			config: &Config{
				URLValidReg: "[",
				ACL:         allowAllACL(),
			},
			wantError: "compile url validation regexp",
		},
		{
			name: "invalid ACL path regexp",
			config: &Config{
				ACL: []ACLRule{{
					Path:        "[",
					Users:       []string{"any"},
					Methods:     []string{"any"},
					ContentType: []string{"any"},
				}},
			},
			wantError: "compile acl[0] path regexp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			controller, err := NewSimpleAccessController(tt.config)

			assert.Nil(t, controller)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantError)
		})
	}
}

func TestNewSimpleAccessController_AllowsMatchingRequest(t *testing.T) {
	t.Parallel()

	controller, err := NewSimpleAccessController(&Config{ACL: allowAllACL()})
	require.NoError(t, err)

	err = controller.Validate(&AuthRequest{
		AuthURL:     "/topics/example",
		AuthUser:    "alice",
		Method:      "POST",
		ContentType: "application/json",
	})
	assert.NoError(t, err)
}

func allowAllACL() []ACLRule {
	return []ACLRule{{
		Path:        ".*",
		Users:       []string{"any"},
		Methods:     []string{"any"},
		ContentType: []string{"any"},
	}}
}
