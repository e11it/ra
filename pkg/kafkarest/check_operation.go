package kafkarest

import "fmt"

// operationAllowedCheckName — имя чекера в конфиге.
const operationAllowedCheckName = "operation_allowed"

// Известные операции корпстандарта — используются как фолбэк, если в
// конфигурации не задан явный AllowedOperations.
const (
	operationCreate   = "CREATE"
	operationUpdate   = "UPDATE"
	operationUpsert   = "UPSERT"
	operationDelete   = "DELETE"
	operationSnapshot = "SNAPSHOT"
	operationEvent    = "EVENT"
)

// defaultAllowedOperations — каталог из корпстандарта.
var defaultAllowedOperations = []string{
	operationCreate,
	operationUpdate,
	operationUpsert,
	operationDelete,
	operationSnapshot,
	operationEvent,
}

// operationAllowedCheck проверяет, что envelope.meta.operation входит в
// настраиваемый корпоративный каталог.
type operationAllowedCheck struct {
	allowed map[string]struct{}
}

func newOperationAllowedCheck(cfg Config) (RecordChecker, error) {
	list := cfg.AllowedOperations
	if len(list) == 0 {
		list = defaultAllowedOperations
	}
	set := make(map[string]struct{}, len(list))
	for _, op := range list {
		if op == "" {
			continue
		}
		set[op] = struct{}{}
	}
	if len(set) == 0 {
		return nil, fmt.Errorf("operation_allowed: empty allowed operations list")
	}
	return &operationAllowedCheck{allowed: set}, nil
}

// Name реализует RecordChecker.
func (c *operationAllowedCheck) Name() string { return operationAllowedCheckName }

// Check реализует RecordChecker.
func (c *operationAllowedCheck) Check(ctx CheckContext, _ *Record) error {
	if ctx.IsTombstone {
		return nil
	}
	if ctx.Envelope == nil {
		return NewValidationError(ctx.Index, c.Name(), "envelope.meta is missing")
	}
	op := ctx.Envelope.Operation
	if op == "" {
		return NewValidationError(ctx.Index, c.Name(), "envelope.meta.operation is empty")
	}
	if _, ok := c.allowed[op]; !ok {
		return NewValidationError(ctx.Index, c.Name(),
			fmt.Sprintf("operation %q is not in allowed catalog", op))
	}
	return nil
}
