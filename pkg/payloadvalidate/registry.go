package payloadvalidate

import "github.com/e11it/ra/pkg/validate"

// NewValidatorFromConfig builds a validator using global validate registry.
func NewValidatorFromConfig(cfg validate.Config) (validate.BodyValidator, error) {
	if cfg.IsEmpty() {
		return nil, nil
	}
	checkers, err := validate.BuildCheckers(cfg)
	if err != nil {
		return nil, err
	}
	if len(checkers) == 0 {
		return nil, nil
	}
	return NewValidator(checkers), nil
}
