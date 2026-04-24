//go:build !company

package ra

import "github.com/e11it/ra/pkg/validate"

// createBodyValidator keeps public builds validation-free even with body_validation in YAML.
func createBodyValidator(_ validate.Config) (validate.BodyValidator, error) {
	return nil, nil
}
