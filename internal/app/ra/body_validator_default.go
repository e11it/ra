//go:build !company

package ra

import (
	"errors"

	"github.com/e11it/ra/pkg/validate"
)

// createBodyValidator rejects company-only body validation in public builds.
func createBodyValidator(cfg validate.Config) (validate.BodyValidator, error) {
	if cfg.Enabled {
		return nil, errors.New("body validation requires company build")
	}
	return nil, nil
}
