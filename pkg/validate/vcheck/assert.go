package vcheck

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/e11it/ra/pkg/validate"
)

var ianaZoneValidity sync.Map // map[string]bool

// PathJoin appends field to a dotted path.
func PathJoin(base, field string) string {
	if base == "" {
		return field
	}
	if field == "" {
		return base
	}
	return base + "." + field
}

// PathIndex builds records[index] prefixed path.
func PathIndex(index int, suffix string) string {
	base := fmt.Sprintf("records[%d]", index)
	if suffix == "" {
		return base
	}
	return base + "." + suffix
}

// RequireField checks object field existence.
func RequireField(rep *validate.Report, rec int, path string, obj map[string]any, field string) (any, bool) {
	if obj == nil {
		rep.AddError(rec, path, "invalid_type", "expected object")
		return nil, false
	}
	value, ok := obj[field]
	if !ok {
		rep.AddError(rec, PathJoin(path, field), "missing_field", "field is required")
		return nil, false
	}
	return value, true
}

// AsObject checks JSON object type.
func AsObject(rep *validate.Report, rec int, path string, value any) (map[string]any, bool) {
	obj, ok := value.(map[string]any)
	if !ok {
		rep.AddError(rec, path, "invalid_type", "expected object")
		return nil, false
	}
	return obj, true
}

// AsString checks JSON string type.
func AsString(rep *validate.Report, rec int, path string, value any) (string, bool) {
	s, ok := value.(string)
	if !ok {
		rep.AddError(rec, path, "invalid_type", "expected string")
		return "", false
	}
	return s, true
}

// AsInt64 checks integer values decoded as float64.
func AsInt64(rep *validate.Report, rec int, path string, value any) (int64, bool) {
	switch n := value.(type) {
	case float64:
		if math.IsNaN(n) || math.IsInf(n, 0) || math.Trunc(n) != n {
			rep.AddError(rec, path, "invalid_type", "expected integer")
			return 0, false
		}
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	default:
		rep.AddError(rec, path, "invalid_type", "expected integer")
		return 0, false
	}
}

// AsTimestampMicros checks timestamp-micros values.
// By default accepts integer JSON values; in extended mode also accepts RFC3339 strings.
func AsTimestampMicros(rep *validate.Report, rec int, path string, value any, extended bool) (int64, bool) {
	if !extended {
		return AsInt64(rep, rec, path, value)
	}

	if n, ok := AsInt64(validate.NewReport(), rec, path, value); ok {
		return n, true
	}

	s, ok := value.(string)
	if !ok {
		rep.AddError(rec, path, "invalid_type", "expected integer or RFC3339 timestamp string")
		return 0, false
	}

	ts, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		ts, err = time.Parse(time.RFC3339, s)
		if err != nil {
			rep.AddError(rec, path, "invalid_type", "expected integer or RFC3339 timestamp string")
			return 0, false
		}
	}

	return ts.UnixMicro(), true
}

// OneOf validates catalog membership.
func OneOf(rep *validate.Report, rec int, path, value string, allowed map[string]struct{}) bool {
	if _, ok := allowed[value]; ok {
		return true
	}
	rep.AddError(rec, path, "not_in_catalog", fmt.Sprintf("value %q is not allowed", value))
	return false
}

// IsUUIDCanonical validates lower-case canonical UUID string.
func IsUUIDCanonical(rep *validate.Report, rec int, path, value string) bool {
	if value == "" {
		rep.AddError(rec, path, "invalid_format", "uuid must not be empty")
		return false
	}
	if value != strings.ToLower(value) {
		rep.AddError(rec, path, "invalid_format", "uuid must be lowercase canonical form")
		return false
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		rep.AddError(rec, path, "invalid_format", "invalid uuid")
		return false
	}
	if parsed.String() != value {
		rep.AddError(rec, path, "invalid_format", "uuid must be canonical form")
		return false
	}
	return true
}

// IsIANAZone validates IANA timezone id.
func IsIANAZone(rep *validate.Report, rec int, path, value string) bool {
	if value == "" {
		rep.AddError(rec, path, "missing_field", "timezone is required")
		return false
	}
	if cached, ok := ianaZoneValidity.Load(value); ok {
		if valid, _ := cached.(bool); valid {
			return true
		}
		rep.AddError(rec, path, "invalid_format", "timezone must be valid IANA id")
		return false
	}
	_, err := time.LoadLocation(value)
	valid := err == nil
	ianaZoneValidity.Store(value, valid)
	if !valid {
		rep.AddError(rec, path, "invalid_format", "timezone must be valid IANA id")
		return false
	}
	return true
}

// UnionString extracts Avro union [null,string] value.
func UnionString(rep *validate.Report, rec int, path string, value any) (string, bool, bool) {
	if value == nil {
		return "", true, true
	}
	obj, ok := value.(map[string]any)
	if !ok {
		rep.AddError(rec, path, "invalid_type", "expected null or Avro union object")
		return "", false, false
	}
	raw, ok := obj["string"]
	if !ok {
		rep.AddError(rec, path, "invalid_type", "expected Avro union {\"string\":...}")
		return "", false, false
	}
	s, ok := raw.(string)
	if !ok {
		rep.AddError(rec, path, "invalid_type", "expected string")
		return "", false, false
	}
	return s, false, true
}
