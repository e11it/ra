//go:build company

package company

import (
	"bytes"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/e11it/ra/pkg/validate"
	"github.com/e11it/ra/pkg/validate/vcheck"
)

const entityKeyCheckName = "entity_key"

type entityKeyCheck struct{}

func newEntityKeyCheck(_ validate.Config) (validate.RecordChecker, error) {
	return &entityKeyCheck{}, nil
}

func (c *entityKeyCheck) Name() string { return entityKeyCheckName }

func (c *entityKeyCheck) Check(ctx *validate.CheckContext, rec *validate.Record, rep *validate.Report) validate.Control {
	pr, ok := parseRecord(ctx, rep)
	if !ok {
		return validate.StopRecord
	}
	if pr.meta == nil {
		rep.AddError(ctx.Index, vcheck.PathIndex(ctx.Index, "envelope.meta"), "missing_field", "field is required")
		return validate.StopRecord
	}

	entityRaw, ok := vcheck.RequireField(rep, ctx.Index, vcheck.PathIndex(ctx.Index, "envelope.meta"), pr.meta, "entityKey")
	if !ok {
		return validate.StopRecord
	}
	entityKey, ok := vcheck.AsString(rep, ctx.Index, vcheck.PathIndex(ctx.Index, "envelope.meta.entityKey"), entityRaw)
	if !ok {
		return validate.StopRecord
	}

	op := ""
	if opRaw, exists := pr.meta["operation"]; exists {
		op, _ = opRaw.(string)
	}
	key, keyPresent, keyOK := extractRecordKey(rep, ctx.Index, rec)
	if !keyOK {
		return validate.StopRecord
	}

	if op == operationEvent && entityKey == "" && !keyPresent {
		return validate.Continue
	}
	if !keyPresent {
		rep.AddError(ctx.Index, vcheck.PathIndex(ctx.Index, "key"), "missing_field", "record key is required")
		return validate.StopRecord
	}
	if key != entityKey {
		rep.AddError(
			ctx.Index,
			vcheck.PathIndex(ctx.Index, "key"),
			"key_mismatch",
			fmt.Sprintf("record key %q does not match envelope.meta.entityKey %q", key, entityKey),
		)
	}
	return validate.Continue
}

func extractRecordKey(rep *validate.Report, index int, rec *validate.Record) (string, bool, bool) {
	if rec == nil {
		rep.AddError(index, vcheck.PathIndex(index, "key"), "missing_field", "record key is required")
		return "", false, false
	}
	trimmed := bytes.TrimSpace(rec.Key)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return "", false, true
	}
	var key string
	if err := sonic.Unmarshal(rec.Key, &key); err != nil {
		rep.AddError(index, vcheck.PathIndex(index, "key"), "invalid_type", "record key must be a JSON string")
		return "", false, false
	}
	return key, true, true
}
