// Package payloadvalidate validates Kafka REST Proxy v2 produce body shape
// and delegates record-specific validation to pluggable checkers.
//
// Reference: https://docs.confluent.io/platform/current/kafka-rest/api.html
// Endpoint: POST /topics/{topic_name}
//
// Scope:
//   - Parses JSON body into records[].
//   - Supports checker pipelines from pkg/validate registry.
//   - Ignores top-level schema fields key_schema*, value_schema*.
//
// Out of scope:
//   - Binary embedded format (application/vnd.kafka.binary.v2+json).
//   - Content-Type validation (handled by ingress/proxy layer).
package payloadvalidate
