//go:build company

package company

import (
	"github.com/e11it/ra/pkg/validate"
	"github.com/e11it/ra/pkg/validate/vcheck"
)

const noPartitionCheckName = "no_partition"

type noPartitionCheck struct{}

func newNoPartitionCheck(_ validate.Config) (validate.RecordChecker, error) {
	return &noPartitionCheck{}, nil
}

func (c *noPartitionCheck) Name() string { return noPartitionCheckName }

func (c *noPartitionCheck) Check(ctx *validate.CheckContext, rec *validate.Record, rep *validate.Report) validate.Control {
	if rec != nil && rec.Partition != nil {
		rep.AddError(
			ctx.Index,
			vcheck.PathIndex(ctx.Index, "partition"),
			"partition_forbidden",
			"record partition must not be set explicitly",
		)
	}
	return validate.Continue
}
