package validate

import (
	"fmt"
	"sort"
	"sync"
)

// Registry stores checker factories by name.
type Registry struct {
	mu        sync.RWMutex
	factories map[string]CheckerFactory
}

// NewRegistry returns an empty checker registry.
func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]CheckerFactory)}
}

// Register adds a checker factory by stable checker name.
func (r *Registry) Register(name string, factory CheckerFactory) error {
	if name == "" {
		return fmt.Errorf("validate: checker name is empty")
	}
	if factory == nil {
		return fmt.Errorf("validate: checker %q has nil factory", name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("validate: checker %q already registered", name)
	}
	r.factories[name] = factory
	return nil
}

// MustRegister is Register that panics on error.
func (r *Registry) MustRegister(name string, factory CheckerFactory) {
	if err := r.Register(name, factory); err != nil {
		panic(err)
	}
}

// Build creates checker instances in config order.
func (r *Registry) Build(cfg Config) ([]RecordChecker, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	checkers := make([]RecordChecker, 0, len(cfg.Checks))
	for _, name := range cfg.Checks {
		factory, ok := r.factories[name]
		if !ok {
			return nil, fmt.Errorf("validate: unknown check %q (available: %v)", name, r.namesUnsafe())
		}
		ch, err := factory(cfg)
		if err != nil {
			return nil, fmt.Errorf("validate: build check %q: %w", name, err)
		}
		checkers = append(checkers, ch)
	}
	return checkers, nil
}

// Names returns registered checker names sorted lexicographically.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.namesUnsafe()
}

func (r *Registry) namesUnsafe() []string {
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

var defaultRegistry = NewRegistry()

// Register adds a checker to package-level default registry.
func Register(name string, factory CheckerFactory) error {
	return defaultRegistry.Register(name, factory)
}

// MustRegister adds a checker to default registry and panics on error.
func MustRegister(name string, factory CheckerFactory) {
	defaultRegistry.MustRegister(name, factory)
}

// BuildCheckers builds checkers from package-level default registry.
func BuildCheckers(cfg Config) ([]RecordChecker, error) {
	return defaultRegistry.Build(cfg)
}

// RegisteredNames returns checker names from default registry.
func RegisteredNames() []string {
	return defaultRegistry.Names()
}
