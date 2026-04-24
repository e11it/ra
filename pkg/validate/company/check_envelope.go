//go:build company

package company

import (
	"fmt"

	"github.com/e11it/ra/pkg/validate"
	"github.com/e11it/ra/pkg/validate/vcheck"
)

const envelopeCheckName = "envelope"

type envelopeCheck struct {
	allowed map[string]struct{}
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
	return &envelopeCheck{allowed: set}, nil
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
	metaObj, ok := vcheck.AsObject(rep, ctx.Index, vcheck.PathJoin(vcheck.PathIndex(ctx.Index, "envelope"), "meta"), metaRaw)
	if ok {
		c.checkMeta(ctx, rep, metaObj)
	}

	techRaw, ok := vcheck.RequireField(rep, ctx.Index, vcheck.PathIndex(ctx.Index, "envelope"), envelopeObj, "tech")
	if !ok {
		return validate.StopRecord
	}
	techObj, ok := vcheck.AsObject(rep, ctx.Index, vcheck.PathJoin(vcheck.PathIndex(ctx.Index, "envelope"), "tech"), techRaw)
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
		vcheck.AsInt64(rep, ctx.Index, vcheck.PathJoin(base, "eventTime"), eventTimeRaw)
	}

	timeZoneRaw, ok := vcheck.RequireField(rep, ctx.Index, base, meta, "eventTimeZone")
	if ok {
		if tz, ok := vcheck.AsString(rep, ctx.Index, vcheck.PathJoin(base, "eventTimeZone"), timeZoneRaw); ok {
			vcheck.IsIANAZone(rep, ctx.Index, vcheck.PathJoin(base, "eventTimeZone"), tz)
		}
	}

	if businessKeyRaw, exists := meta["businessKey"]; exists {
		vcheck.UnionString(rep, ctx.Index, vcheck.PathJoin(base, "businessKey"), businessKeyRaw)
	}
	if businessDateRaw, exists := meta["businessDate"]; exists {
		vcheck.UnionInt(rep, ctx.Index, vcheck.PathJoin(base, "businessDate"), businessDateRaw)
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
		if schemaVersion, ok := vcheck.AsString(rep, ctx.Index, vcheck.PathJoin(base, "schemaVersion"), schemaVersionRaw); ok {
			vcheck.IsSemverLike(rep, ctx.Index, vcheck.PathJoin(base, "schemaVersion"), schemaVersion)
		}
	}

	producedAtRaw, ok := vcheck.RequireField(rep, ctx.Index, base, tech, "producedAt")
	if ok {
		vcheck.AsInt64(rep, ctx.Index, vcheck.PathJoin(base, "producedAt"), producedAtRaw)
	}

	if seqRaw, exists := tech["sequence"]; exists {
		vcheck.UnionInt(rep, ctx.Index, vcheck.PathJoin(base, "sequence"), seqRaw)
	}
	if traceRaw, exists := tech["traceId"]; exists {
		vcheck.UnionString(rep, ctx.Index, vcheck.PathJoin(base, "traceId"), traceRaw)
	}
}
