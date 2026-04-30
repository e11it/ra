//go:build company

package company

import (
	"github.com/e11it/ra/pkg/validate"
	"github.com/e11it/ra/pkg/validate/vcheck"
)

const payloadCheckName = "payload"

type payloadCheck struct{}

const opMismatchCode = "operation_payload_mismatch"

func newPayloadCheck(_ validate.Config) (validate.RecordChecker, error) {
	return &payloadCheck{}, nil
}

func (c *payloadCheck) Name() string { return payloadCheckName }

func (c *payloadCheck) Check(ctx *validate.CheckContext, _ *validate.Record, rep *validate.Report) validate.Control {
	pr, ok := parseRecord(ctx, rep)
	if !ok {
		return validate.StopRecord
	}

	if pr.payload == nil && pr.before == nil {
		rep.AddError(
			ctx.Index,
			vcheck.PathIndex(ctx.Index, "payload"),
			"invalid_payload_state",
			"payload and payloadBefore cannot both be null",
		)
	}

	op := ""
	if pr.meta != nil {
		if raw, exists := pr.meta["operation"]; exists {
			if s, ok := raw.(string); ok {
				op = s
			}
		}
	}
	c.checkOperationPayload(ctx, rep, op, pr.payload == nil, pr.before != nil)
	return validate.Continue
}

func (c *payloadCheck) checkOperationPayload(
	ctx *validate.CheckContext,
	rep *validate.Report,
	op string,
	payloadNil bool,
	beforePresent bool,
) {
	switch op {
	case operationCreate, operationSnapshot:
		if payloadNil {
			rep.AddWarning(
				ctx.Index,
				vcheck.PathIndex(ctx.Index, "payload"),
				opMismatchCode,
				"payload should be non-null for operation "+op,
			)
		}
	case operationUpdate, operationUpsert:
		if payloadNil {
			rep.AddWarning(
				ctx.Index,
				vcheck.PathIndex(ctx.Index, "payload"),
				opMismatchCode,
				"payload should be non-null for operation "+op,
			)
		}
	case operationDelete:
		if !payloadNil {
			rep.AddWarning(
				ctx.Index,
				vcheck.PathIndex(ctx.Index, "payload"),
				opMismatchCode,
				"payload should be null for DELETE",
			)
		}
	case operationEvent:
		if beforePresent {
			rep.AddWarning(
				ctx.Index,
				vcheck.PathIndex(ctx.Index, "payloadBefore"),
				opMismatchCode,
				"payloadBefore should be null for EVENT",
			)
		}
	}
}
