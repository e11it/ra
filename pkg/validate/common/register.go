package common

import "github.com/e11it/ra/pkg/validate"

func init() {
	validate.MustRegister(isTombstoneCheckName, newIsTombstoneCheck)
}
