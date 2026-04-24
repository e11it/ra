package validate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/bytedance/sonic"
)

// ValueStore provides lazy decoding and shared cache for record value.
type ValueStore struct {
	raw json.RawMessage

	onceRoot sync.Once
	root     map[string]any
	rootErr  error

	mu    sync.Mutex
	cache map[string]any
}

// NewValueStore initializes lazy record value state.
func NewValueStore(raw json.RawMessage) *ValueStore {
	return &ValueStore{
		raw:   raw,
		cache: make(map[string]any),
	}
}

// Raw returns original record value bytes.
func (v *ValueStore) Raw() json.RawMessage {
	if v == nil {
		return nil
	}
	return v.raw
}

// IsNull reports whether value is absent or json null.
func (v *ValueStore) IsNull() bool {
	if v == nil {
		return true
	}
	trimmed := bytes.TrimSpace(v.raw)
	if len(trimmed) == 0 {
		return true
	}
	return bytes.Equal(trimmed, []byte("null"))
}

// Root lazily decodes value as JSON object.
func (v *ValueStore) Root() (map[string]any, bool, error) {
	if v == nil || v.IsNull() {
		return nil, false, nil
	}
	v.onceRoot.Do(func() {
		var root map[string]any
		if err := sonic.Unmarshal(v.raw, &root); err != nil {
			v.rootErr = fmt.Errorf("decode value root: %w", err)
			return
		}
		v.root = root
	})
	if v.rootErr != nil {
		return nil, true, v.rootErr
	}
	return v.root, true, nil
}

// Cached memoizes arbitrary derived values by key.
func (v *ValueStore) Cached(key string, build func() (any, error)) (any, error) {
	if v == nil {
		return nil, fmt.Errorf("value store is nil")
	}
	v.mu.Lock()
	val, ok := v.cache[key]
	v.mu.Unlock()
	if ok {
		return val, nil
	}
	built, err := build()
	if err != nil {
		return nil, err
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if existing, ok := v.cache[key]; ok {
		return existing, nil
	}
	v.cache[key] = built
	return built, nil
}
