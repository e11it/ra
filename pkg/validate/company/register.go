//go:build company

package company

import "github.com/e11it/ra/pkg/validate"

func init() {
	validate.MustRegister(noPartitionCheckName, newNoPartitionCheck)
	validate.MustRegister(envelopeCheckName, newEnvelopeCheck)
	validate.MustRegister(payloadCheckName, newPayloadCheck)
	validate.MustRegister(entityKeyCheckName, newEntityKeyCheck)
	validate.MustRegister(corporateV1CheckName, newCorporateV1Check)
}
