//go:build company

package company

import "github.com/e11it/ra/pkg/validate"

const corporateV1CheckName = "corporate_v1"

type corporateV1Check struct {
	checks []validate.RecordChecker
}

func newCorporateV1Check(cfg validate.Config) (validate.RecordChecker, error) {
	noPartition, err := newNoPartitionCheck(cfg)
	if err != nil {
		return nil, err
	}
	envelope, err := newEnvelopeCheck(cfg)
	if err != nil {
		return nil, err
	}
	payload, err := newPayloadCheck(cfg)
	if err != nil {
		return nil, err
	}
	entityKey, err := newEntityKeyCheck(cfg)
	if err != nil {
		return nil, err
	}
	return &corporateV1Check{
		checks: []validate.RecordChecker{noPartition, envelope, payload, entityKey},
	}, nil
}

func (c *corporateV1Check) Name() string { return corporateV1CheckName }

func (c *corporateV1Check) Check(ctx *validate.CheckContext, rec *validate.Record, rep *validate.Report) validate.Control {
	for _, check := range c.checks {
		ctrl := check.Check(ctx, rec, rep)
		if ctrl != validate.Continue {
			return ctrl
		}
	}
	return validate.Continue
}
