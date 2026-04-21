package kafkarest

// BodyValidator проверяет тело REST v2 produce-запроса к Kafka REST Proxy.
//
// Реализация должна уметь безопасно обрабатывать:
//   - пустое тело: валидатор возвращает nil (нечего валидировать);
//   - tombstone-записи ({"key": ..., "value": null}) — пропускаются без проверок;
//   - остальные записи — прогоняются через цепочку RecordChecker'ов.
type BodyValidator interface {
	Validate(body []byte) error
}

// RecordChecker — одна проверка одной записи в массиве records[].
//
// Контракт:
//   - Реализация должна быть stateless и безопасной для конкурентного использования.
//   - Если запись является tombstone (ctx.IsTombstone == true), реализация ОБЯЗАНА
//     вернуть nil немедленно — tombstone не содержит envelope и не подлежит
//     семантическим проверкам.
//   - Ошибку возвращать через NewValidationError, чтобы в middleware была
//     консистентная формулировка.
type RecordChecker interface {
	// Name — стабильный идентификатор чекера, используемый в конфиге и в
	// сообщениях об ошибках. Например: "entity_key_match".
	Name() string

	// Check проверяет запись rec в контексте ctx.
	Check(ctx CheckContext, rec *Record) error
}

// CheckerFactory строит экземпляр RecordChecker по конфигурации валидатора.
// Используется реестром чекеров в registry.go.
type CheckerFactory func(cfg Config) (RecordChecker, error)
