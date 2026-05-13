//go:build company

package company

import (
	"fmt"

	"github.com/e11it/ra/pkg/validate"
	"github.com/e11it/ra/pkg/validate/vcheck"
)

const envelopeCheckName = "envelope"

type envelopeCheck struct {
	allowed           map[string]struct{}
	extendedAvroTypes bool
}

func newEnvelopeCheck(cfg validate.Config) (validate.RecordChecker, error) {
	allowed := cfg.List("allowed_operations")
	if len(allowed) == 0 {
		allowed = defaultAllowedOperations
	}
	set := make(map[string]struct{}, len(allowed))
	for _, item := range allowed {
		if item == "" {
			continue
		}
		set[item] = struct{}{}
	}
	if len(set) == 0 {
		return nil, fmt.Errorf("envelope: allowed operations is empty")
	}
	return &envelopeCheck{
		allowed:           set,
		extendedAvroTypes: cfg.ExtendedAvroTypes,
	}, nil
}

func (c *envelopeCheck) Name() string { return envelopeCheckName }

func (c *envelopeCheck) Check(ctx *validate.CheckContext, _ *validate.Record, rep *validate.Report) validate.Control {
	pr, ok := parseRecord(ctx, rep)
	if !ok {
		return validate.StopRecord
	}

	envelopeRaw, exists := pr.root["envelope"]
	if !exists {
		rep.AddError(ctx.Index, vcheck.PathIndex(ctx.Index, "envelope"), "missing_field", "field is required")
		return validate.StopRecord
	}
	envelopeObj, ok := vcheck.AsObject(rep, ctx.Index, vcheck.PathIndex(ctx.Index, "envelope"), envelopeRaw)
	if !ok {
		return validate.StopRecord
	}

	metaRaw, ok := vcheck.RequireField(rep, ctx.Index, vcheck.PathIndex(ctx.Index, "envelope"), envelopeObj, "meta")
	if !ok {
		return validate.StopRecord
	}
	envelopePath := vcheck.PathIndex(ctx.Index, "envelope")
	metaPath := vcheck.PathJoin(envelopePath, "meta")
	metaObj, ok := vcheck.AsObject(rep, ctx.Index, metaPath, metaRaw)
	if ok {
		c.checkMeta(ctx, rep, metaObj)
	}

	techRaw, ok := vcheck.RequireField(rep, ctx.Index, vcheck.PathIndex(ctx.Index, "envelope"), envelopeObj, "tech")
	if !ok {
		return validate.StopRecord
	}
	techPath := vcheck.PathJoin(envelopePath, "tech")
	techObj, ok := vcheck.AsObject(rep, ctx.Index, techPath, techRaw)
	if ok {
		c.checkTech(ctx, rep, techObj)
	}
	return validate.Continue
}

func (c *envelopeCheck) checkMeta(ctx *validate.CheckContext, rep *validate.Report, meta map[string]any) {
	base := vcheck.PathJoin(vcheck.PathIndex(ctx.Index, "envelope"), "meta")

	eventIDRaw, ok := vcheck.RequireField(rep, ctx.Index, base, meta, "eventId")
	if ok {
		if s, ok := vcheck.AsString(rep, ctx.Index, vcheck.PathJoin(base, "eventId"), eventIDRaw); ok {
			vcheck.IsUUIDCanonical(rep, ctx.Index, vcheck.PathJoin(base, "eventId"), s)
		}
	}

	entityKeyRaw, ok := vcheck.RequireField(rep, ctx.Index, base, meta, "entityKey")
	if ok {
		vcheck.AsString(rep, ctx.Index, vcheck.PathJoin(base, "entityKey"), entityKeyRaw)
	}

	opRaw, ok := vcheck.RequireField(rep, ctx.Index, base, meta, "operation")
	if ok {
		if op, ok := vcheck.AsString(rep, ctx.Index, vcheck.PathJoin(base, "operation"), opRaw); ok {
			vcheck.OneOf(rep, ctx.Index, vcheck.PathJoin(base, "operation"), op, c.allowed)
		}
	}

	eventTimeRaw, ok := vcheck.RequireField(rep, ctx.Index, base, meta, "eventTime")
	if ok {
		vcheck.AsTimestampMillis(rep, ctx.Index, vcheck.PathJoin(base, "eventTime"), eventTimeRaw, c.extendedAvroTypes)
	}

	timeZoneRaw, ok := vcheck.RequireField(rep, ctx.Index, base, meta, "eventTimeZone")
	if ok {
		if tz, ok := vcheck.AsString(rep, ctx.Index, vcheck.PathJoin(base, "eventTimeZone"), timeZoneRaw); ok {
			vcheck.IsIANAZone(rep, ctx.Index, vcheck.PathJoin(base, "eventTimeZone"), tz)
		}
	}
}

func (c *envelopeCheck) checkTech(ctx *validate.CheckContext, rep *validate.Report, tech map[string]any) {
	base := vcheck.PathJoin(vcheck.PathIndex(ctx.Index, "envelope"), "tech")

	sourceRaw, ok := vcheck.RequireField(rep, ctx.Index, base, tech, "sourceSystem")
	if ok {
		vcheck.AsString(rep, ctx.Index, vcheck.PathJoin(base, "sourceSystem"), sourceRaw)
	}

	schemaVersionRaw, ok := vcheck.RequireField(rep, ctx.Index, base, tech, "schemaVersion")
	if ok {
		schemaPath := vcheck.PathJoin(base, "schemaVersion")
		if schemaVersion, ok := vcheck.AsInt64(rep, ctx.Index, schemaPath, schemaVersionRaw); ok && schemaVersion != 0 {
			rep.AddError(ctx.Index, schemaPath, "invalid_value", "schemaVersion must be 0")
		}
	}

	producedAtRaw, ok := vcheck.RequireField(rep, ctx.Index, base, tech, "producedAt")
	if ok {
		vcheck.AsTimestampMillis(rep, ctx.Index, vcheck.PathJoin(base, "producedAt"), producedAtRaw, c.extendedAvroTypes)
	}

	if traceRaw, exists := tech["traceId"]; exists {
		tracePath := vcheck.PathJoin(base, "traceId")
		if traceID, isNull, ok := vcheck.UnionString(rep, ctx.Index, tracePath, traceRaw); ok && !isNull {
			vcheck.IsUUIDCanonical(rep, ctx.Index, tracePath, traceID)
		}
	}
}
