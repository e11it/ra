package kafkarest

import (
	"encoding/json"
	"fmt"
)

// ProduceRequest — минимальная модель тела POST /topics/{topic} для Kafka REST v2.
// Другие поля (value_schema, value_schema_id, key_schema*) нас не интересуют —
// мы валидируем только records[].
type ProduceRequest struct {
	Records []Record `json:"records"`
}

// Record — одна запись из массива records[] REST v2 produce-запроса.
// Key/Value оставлены как RawMessage, чтобы:
//   - корректно отличить отсутствующее поле от null;
//   - извлечь envelope без лишних промежуточных структур, когда value — объект.
type Record struct {
	Key       json.RawMessage `json:"key,omitempty"`
	Value     json.RawMessage `json:"value"`
	Partition *int            `json:"partition,omitempty"`
}

// Envelope — корпоративный envelope сообщения (см. docs/Корпоративный стандарт ...).
// Распарсивается из records[i].value, если это объект.
type Envelope struct {
	Meta EventMeta `json:"envelope"`
}

// envelopeWrapper используется для декодирования двухуровневой вложенности
// records[i].value.envelope.meta. Для JSON encoding Avro required-поля
// entityKey/operation — это плоские строки, поэтому мы не обрабатываем union'ы.
type envelopeWrapper struct {
	Envelope struct {
		Meta EventMeta `json:"meta"`
	} `json:"envelope"`
}

// EventMeta — плоская проекция envelope.meta, достаточная для текущих чекеров.
// Необязательные поля (businessKey, businessDate и т.п.) не парсим, чтобы не
// зависеть от формы union'ов в Avro JSON encoding.
type EventMeta struct {
	EntityKey     string `json:"entityKey"`
	Operation     string `json:"operation"`
	EventTimeZone string `json:"eventTimeZone"`
}

// CheckContext — контекст выполнения одной проверки.
type CheckContext struct {
	// Index — позиция записи в records[].
	Index int

	// Envelope — распарсенный envelope для non-tombstone записей. Для tombstone — nil.
	Envelope *EventMeta

	// IsTombstone — true, если запись является маркером удаления (value == null).
	// Чекеры обязаны пропускать такие записи (возвращать nil).
	IsTombstone bool
}

// ValidationError — ошибка валидации одной записи.
// Форматируется в единый вид для middleware и логов.
type ValidationError struct {
	// Index — номер записи в records[], к которой относится ошибка.
	Index int
	// Check — имя чекера, инициировавшего ошибку. Пустая строка — ошибка до чекеров
	// (например, невалидный JSON или отсутствие records[]).
	Check string
	// Reason — человеко-читаемое объяснение причины.
	Reason string
	// Err — исходная ошибка для wrapping/Unwrap, если есть.
	Err error
}

// Error реализует error.
func (e *ValidationError) Error() string {
	switch {
	case e.Check != "" && e.Err != nil:
		return fmt.Sprintf("records[%d] %s: %s: %v", e.Index, e.Check, e.Reason, e.Err)
	case e.Check != "":
		return fmt.Sprintf("records[%d] %s: %s", e.Index, e.Check, e.Reason)
	case e.Err != nil:
		return fmt.Sprintf("%s: %v", e.Reason, e.Err)
	default:
		return e.Reason
	}
}

// Unwrap возвращает исходную ошибку, если есть.
func (e *ValidationError) Unwrap() error { return e.Err }

// NewValidationError конструирует ошибку валидации для записи с указанным чекером.
func NewValidationError(index int, check, reason string) *ValidationError {
	return &ValidationError{Index: index, Check: check, Reason: reason}
}

// NewRequestError конструирует ошибку валидации уровня всего запроса
// (не относящуюся к конкретной записи) — например, невалидный JSON.
func NewRequestError(reason string, err error) *ValidationError {
	return &ValidationError{Index: -1, Reason: reason, Err: err}
}
