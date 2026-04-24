package validate

import "encoding/json"

// ProduceRequest is a minimal Kafka REST v2 produce body model.
type ProduceRequest struct {
	KeySchema     json.RawMessage `json:"key_schema,omitempty"`
	KeySchemaID   json.RawMessage `json:"key_schema_id,omitempty"`
	ValueSchema   json.RawMessage `json:"value_schema,omitempty"`
	ValueSchemaID json.RawMessage `json:"value_schema_id,omitempty"`
	Records       []Record        `json:"records"`
}

// Record is one item from records[].
type Record struct {
	Key       json.RawMessage `json:"key,omitempty"`
	Value     json.RawMessage `json:"value"`
	Partition *int            `json:"partition,omitempty"`
}
