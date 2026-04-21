package kafkarest

// Config — конфигурация валидатора тела Kafka REST v2 produce-запросов.
//
// Поле маппится из ra.Config.BodyValidation через адаптер в internal/app/ra.
type Config struct {
	// Enabled — если false, валидатор не создаётся и middleware не читает тело.
	Enabled bool

	// AllowedOperations — каталог допустимых значений envelope.meta.operation.
	// Сравнение регистрозависимое (стандарт требует UPPERCASE).
	AllowedOperations []string

	// Checks — список имён RecordChecker'ов, которые должны быть активированы.
	// Порядок имеет значение: чекеры выполняются в указанной последовательности,
	// первая ошибка прерывает обработку записи.
	Checks []string
}

// IsEmpty возвращает true, если конфиг фактически выключает валидацию —
// либо флагом Enabled, либо отсутствием чекеров.
func (c Config) IsEmpty() bool {
	return !c.Enabled || len(c.Checks) == 0
}
