package common

import (
	"fmt"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/e11it/ra/pkg/validate"
)

const isTombstoneCheckName = "is_tombstone"

type isTombstoneCheck struct{}

func newIsTombstoneCheck(_ validate.Config) (validate.RecordChecker, error) {
	return &isTombstoneCheck{}, nil
}

func (c *isTombstoneCheck) Name() string { return isTombstoneCheckName }

func (c *isTombstoneCheck) Check(ctx *validate.CheckContext, rec *validate.Record, rep *validate.Report) validate.Control {
	if ctx == nil || ctx.Values == nil || !ctx.Values.IsPresent() || !ctx.Values.IsNull() {
		return validate.Continue
	}

	var key string
	if rec == nil || len(rec.Key) == 0 || sonic.Unmarshal(rec.Key, &key) != nil || strings.TrimSpace(key) == "" {
		if rep != nil {
			rep.AddError(
				ctx.Index,
				fmt.Sprintf("records[%d].key", ctx.Index),
				"invalid_tombstone_key",
				"tombstone key must be a non-empty string",
			)
		}
		return validate.StopRecord
	}

	return validate.StopRecord
}
