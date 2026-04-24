package common

import "github.com/e11it/ra/pkg/validate"

const isTombstoneCheckName = "is_tombstone"

type isTombstoneCheck struct{}

func newIsTombstoneCheck(_ validate.Config) (validate.RecordChecker, error) {
	return &isTombstoneCheck{}, nil
}

func (c *isTombstoneCheck) Name() string { return isTombstoneCheckName }

func (c *isTombstoneCheck) Check(ctx *validate.CheckContext, _ *validate.Record, _ *validate.Report) validate.Control {
	if ctx == nil || ctx.Values == nil || ctx.Values.IsNull() {
		return validate.StopRecord
	}
	return validate.Continue
}
