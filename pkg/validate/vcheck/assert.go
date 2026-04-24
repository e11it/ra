package vcheck

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/e11it/ra/pkg/validate"
)

var semverLikeRe = regexp.MustCompile(`^[0-9]+(\.[0-9]+){0,2}([\-+][A-Za-z0-9.\-]+)?$`)
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

// IsSemverLike validates simple semver-like values such as "1", "1.2" or "1.2.3".
func IsSemverLike(rep *validate.Report, rec int, path, value string) bool {
	if value == "" {
		rep.AddError(rec, path, "missing_field", "version is required")
		return false
	}
	if semverLikeRe.MatchString(value) {
		return true
	}
	rep.AddError(rec, path, "invalid_format", "version must match semver-like format")
	return false
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

// UnionInt extracts Avro union [null,int] value.
func UnionInt(rep *validate.Report, rec int, path string, value any) (int64, bool, bool) {
	if value == nil {
		return 0, true, true
	}
	obj, ok := value.(map[string]any)
	if !ok {
		rep.AddError(rec, path, "invalid_type", "expected null or Avro union object")
		return 0, false, false
	}
	raw, ok := obj["int"]
	if !ok {
		rep.AddError(rec, path, "invalid_type", "expected Avro union {\"int\":...}")
		return 0, false, false
	}
	n, valid := AsInt64(rep, rec, path, raw)
	if !valid {
		return 0, false, false
	}
	return n, false, true
}
