//go:build company

package company

const (
	operationCreate   = "CREATE"
	operationUpdate   = "UPDATE"
	operationUpsert   = "UPSERT"
	operationDelete   = "DELETE"
	operationSnapshot = "SNAPSHOT"
	operationEvent    = "EVENT"
)

var defaultAllowedOperations = []string{
	operationCreate,
	operationUpdate,
	operationUpsert,
	operationDelete,
	operationSnapshot,
	operationEvent,
}
