package payloadvalidate

import (
	"bytes"

	"github.com/bytedance/sonic"
	"github.com/e11it/ra/pkg/validate"
)

type produceV2Validator struct {
	checkers []validate.RecordChecker
}

// NewValidator builds a body validator from ordered checkers.
func NewValidator(checkers []validate.RecordChecker) validate.BodyValidator {
	return &produceV2Validator{checkers: checkers}
}

func (v *produceV2Validator) Validate(body []byte) *validate.Report {
	rep := validate.NewReport()
	if len(bytes.TrimSpace(body)) == 0 || len(v.checkers) == 0 {
		return rep
	}

	var req validate.ProduceRequest
	if err := sonic.Unmarshal(body, &req); err != nil {
		rep.AddError(-1, "", "malformed_body", "invalid kafka rest produce body")
		return rep
	}
	if len(req.Records) == 0 {
		rep.AddError(-1, "records", "empty_records", "records[] is empty")
		return rep
	}

	for i := range req.Records {
		rec := &req.Records[i]
		ctx := &validate.CheckContext{
			Index:  i,
			Values: validate.NewValueStore(rec.Value),
		}
		for _, c := range v.checkers {
			switch c.Check(ctx, rec, rep) {
			case validate.StopRecord:
				goto nextRecord
			case validate.AbortAll:
				return rep
			}
		}
	nextRecord:
	}
	return rep
}
