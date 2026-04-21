package kafkarest

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// entityKeyMatchCheckName — имя чекера в конфиге.
const entityKeyMatchCheckName = "entity_key_match"

// entityKeyMatchCheck проверяет инвариант корпстандарта:
//
//	Kafka.key (bytes) == UTF-8(envelope.meta.entityKey)
//
// Для operation=EVENT допустимо пустое entityKey и отсутствие key.
type entityKeyMatchCheck struct{}

func newEntityKeyMatchCheck(_ Config) (RecordChecker, error) {
	return &entityKeyMatchCheck{}, nil
}

// Name реализует RecordChecker.
func (c *entityKeyMatchCheck) Name() string { return entityKeyMatchCheckName }

// Check реализует RecordChecker.
func (c *entityKeyMatchCheck) Check(ctx CheckContext, rec *Record) error {
	if ctx.IsTombstone {
		return nil
	}
	if ctx.Envelope == nil {
		return NewValidationError(ctx.Index, c.Name(), "envelope.meta is missing")
	}

	keyStr, keyPresent, err := extractStringKey(rec.Key)
	if err != nil {
		return NewValidationError(ctx.Index, c.Name(),
			fmt.Sprintf("cannot decode record key: %v", err))
	}
	entityKey := ctx.Envelope.EntityKey
	operation := ctx.Envelope.Operation

	// EVENT: допустимо отсутствие и key, и entityKey (или оба пустые).
	if operation == operationEvent && !keyPresent && entityKey == "" {
		return nil
	}

	if !keyPresent {
		return NewValidationError(ctx.Index, c.Name(),
			fmt.Sprintf("record key is missing while envelope.meta.entityKey=%q", entityKey))
	}
	if keyStr != entityKey {
		return NewValidationError(ctx.Index, c.Name(),
			fmt.Sprintf("record key %q does not match envelope.meta.entityKey %q",
				keyStr, entityKey))
	}
	return nil
}

// extractStringKey приводит JSON-представление key к строке.
//
// Поддерживаем два случая:
//  1. key закодирован как JSON-строка: "554123" → 554123;
//  2. key отсутствует (raw пустой или JSON null) → keyPresent=false.
//
// Числовые/булевы/объектные ключи в корпстандарте недопустимы
// (entityKey — строка), для них вернём ошибку декодирования.
func extractStringKey(raw json.RawMessage) (value string, present bool, err error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return "", false, nil
	}
	var s string
	if err := json.Unmarshal(trimmed, &s); err != nil {
		return "", true, fmt.Errorf("expected string key, got %s", string(trimmed))
	}
	return s, true, nil
}
