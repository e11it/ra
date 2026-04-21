package kafkarest

import (
	"bytes"
	"fmt"

	"github.com/bytedance/sonic"
)

// avroV2Validator — реализация BodyValidator для REST v2 Avro produce-запросов.
//
// Парсит тело в ProduceRequest, для каждой записи:
//   - распознаёт tombstone (value пустой или JSON null) и пропускает проверки;
//   - иначе извлекает envelope.meta и прогоняет чекеры по порядку.
type avroV2Validator struct {
	checkers []RecordChecker
}

// NewValidator создаёт BodyValidator из списка чекеров.
// Пустой список чекеров означает «валидация по сути отключена»: Validate всегда вернёт nil.
func NewValidator(checkers []RecordChecker) BodyValidator {
	return &avroV2Validator{checkers: checkers}
}

// Validate реализует BodyValidator.
//
// Ошибки:
//   - пустое тело — nil (нечего валидировать);
//   - невалидный JSON, отсутствие/пустой records[] — *ValidationError;
//   - нарушение конкретного чекера — *ValidationError с указанием индекса записи.
func (v *avroV2Validator) Validate(body []byte) error {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	if len(v.checkers) == 0 {
		return nil
	}

	var req ProduceRequest
	if err := sonic.Unmarshal(body, &req); err != nil {
		return NewRequestError("invalid kafka rest produce body: malformed json", err)
	}
	if len(req.Records) == 0 {
		return NewRequestError("invalid kafka rest produce body: records[] is empty", nil)
	}

	for i := range req.Records {
		rec := &req.Records[i]
		ctx, err := v.buildContext(i, rec)
		if err != nil {
			return err
		}
		if ctx.IsTombstone {
			continue
		}
		for _, c := range v.checkers {
			if err := c.Check(ctx, rec); err != nil {
				return err
			}
		}
	}
	return nil
}

// buildContext распознаёт tombstone и извлекает envelope.meta для остальных записей.
func (v *avroV2Validator) buildContext(index int, rec *Record) (CheckContext, error) {
	if isTombstone(rec.Value) {
		return CheckContext{Index: index, IsTombstone: true}, nil
	}

	var w envelopeWrapper
	if err := sonic.Unmarshal(rec.Value, &w); err != nil {
		return CheckContext{}, NewValidationError(index, "",
			fmt.Sprintf("cannot decode records[%d].value as envelope: %v", index, err))
	}
	meta := w.Envelope.Meta

	return CheckContext{Index: index, Envelope: &meta}, nil
}

// isTombstone возвращает true, если значение записи отсутствует или является JSON null.
func isTombstone(value []byte) bool {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) == 0 {
		return true
	}
	return bytes.Equal(trimmed, []byte("null"))
}
