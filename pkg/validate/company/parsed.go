//go:build company

package company

import (
	"fmt"

	"github.com/e11it/ra/pkg/validate"
	"github.com/e11it/ra/pkg/validate/vcheck"
)

type parsedRecord struct {
	root    map[string]any
	meta    map[string]any
	tech    map[string]any
	payload any
	before  any
}

func parseRecord(ctx *validate.CheckContext, rep *validate.Report) (*parsedRecord, bool) {
	if ctx == nil || ctx.Values == nil {
		rep.AddError(-1, "", "missing_value", "record value is missing")
		return nil, false
	}

	cached, err := ctx.Values.Cached("company.parsed", func() (any, error) {
		root, ok, rootErr := ctx.Values.Root()
		if rootErr != nil {
			return nil, rootErr
		}
		if !ok {
			return nil, nil
		}
		pr := &parsedRecord{
			root:    root,
			payload: root["payload"],
			before:  root["payloadBefore"],
		}
		envelopeRaw, exists := root["envelope"]
		if !exists {
			return pr, nil
		}
		envelopeObj, ok := envelopeRaw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("envelope is not object")
		}
		if meta, ok := envelopeObj["meta"].(map[string]any); ok {
			pr.meta = meta
		}
		if tech, ok := envelopeObj["tech"].(map[string]any); ok {
			pr.tech = tech
		}
		return pr, nil
	})
	if err != nil {
		rep.AddError(ctx.Index, vcheck.PathIndex(ctx.Index, "value"), "decode_error", err.Error())
		return nil, false
	}
	if cached == nil {
		rep.AddError(ctx.Index, vcheck.PathIndex(ctx.Index, "value"), "missing_value", "record value is null")
		return nil, false
	}
	pr, ok := cached.(*parsedRecord)
	if !ok {
		rep.AddError(ctx.Index, vcheck.PathIndex(ctx.Index, "value"), "decode_error", "unexpected parsed payload type")
		return nil, false
	}
	return pr, true
}
