//go:build company

package ra

import (
	"github.com/e11it/ra/pkg/payloadvalidate"
	"github.com/e11it/ra/pkg/validate"
	_ "github.com/e11it/ra/pkg/validate/common"
	_ "github.com/e11it/ra/pkg/validate/company"
)

func createBodyValidator(cfg validate.Config) (validate.BodyValidator, error) {
	return payloadvalidate.NewValidatorFromConfig(cfg)
}
