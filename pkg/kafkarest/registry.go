package kafkarest

import "fmt"

// defaultRegistry — встроенный набор чекеров.
// Новый чекер = добавить запись в эту map + реализовать CheckerFactory.
var defaultRegistry = map[string]CheckerFactory{
	"entity_key_match":   newEntityKeyMatchCheck,
	"operation_allowed":  newOperationAllowedCheck,
}

// BuildCheckers собирает список RecordChecker'ов по именам, указанным в cfg.Checks.
// Порядок соответствует порядку в cfg.Checks. Неизвестное имя — ошибка.
func BuildCheckers(cfg Config) ([]RecordChecker, error) {
	checkers := make([]RecordChecker, 0, len(cfg.Checks))
	for _, name := range cfg.Checks {
		factory, ok := defaultRegistry[name]
		if !ok {
			return nil, fmt.Errorf("kafkarest: unknown check %q", name)
		}
		ch, err := factory(cfg)
		if err != nil {
			return nil, fmt.Errorf("kafkarest: build check %q: %w", name, err)
		}
		checkers = append(checkers, ch)
	}
	return checkers, nil
}

// NewValidatorFromConfig — удобный конструктор: строит чекеры по cfg и возвращает
// BodyValidator. Если cfg.IsEmpty() — возвращает nil валидатор (значит,
// middleware не должен читать тело).
func NewValidatorFromConfig(cfg Config) (BodyValidator, error) {
	if cfg.IsEmpty() {
		return nil, nil
	}
	checkers, err := BuildCheckers(cfg)
	if err != nil {
		return nil, err
	}
	if len(checkers) == 0 {
		return nil, nil
	}
	return NewValidator(checkers), nil
}
