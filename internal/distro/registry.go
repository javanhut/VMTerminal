package distro

import (
	"fmt"
	"sync"
)

var (
	registry     = make(map[ID]Provider)
	registryLock sync.RWMutex
	defaultID    ID = Alpine
)

// Register adds a provider to the registry.
// This should be called from init() functions in provider implementations.
func Register(p Provider) {
	registryLock.Lock()
	defer registryLock.Unlock()
	registry[p.ID()] = p
}

// Get returns a provider by ID.
func Get(id ID) (Provider, error) {
	registryLock.RLock()
	defer registryLock.RUnlock()

	p, ok := registry[id]
	if !ok {
		return nil, &ErrUnknownDistro{ID: id}
	}
	return p, nil
}

// GetDefault returns the default distribution provider.
func GetDefault() (Provider, error) {
	return Get(defaultID)
}

// DefaultID returns the default distribution ID.
func DefaultID() ID {
	return defaultID
}

// SetDefault changes the default distribution ID.
func SetDefault(id ID) error {
	registryLock.Lock()
	defer registryLock.Unlock()

	if _, ok := registry[id]; !ok {
		return &ErrUnknownDistro{ID: id}
	}
	defaultID = id
	return nil
}

// List returns all registered provider IDs.
func List() []ID {
	registryLock.RLock()
	defer registryLock.RUnlock()

	ids := make([]ID, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}
	return ids
}

// ListProviders returns all registered providers.
func ListProviders() []Provider {
	registryLock.RLock()
	defer registryLock.RUnlock()

	providers := make([]Provider, 0, len(registry))
	for _, p := range registry {
		providers = append(providers, p)
	}
	return providers
}

// IsRegistered checks if a distribution ID is registered.
func IsRegistered(id ID) bool {
	registryLock.RLock()
	defer registryLock.RUnlock()
	_, ok := registry[id]
	return ok
}

// ParseID parses a string into a distro ID, returning an error if unknown.
func ParseID(s string) (ID, error) {
	id := ID(s)
	if !IsRegistered(id) {
		return "", &ErrUnknownDistro{ID: id}
	}
	return id, nil
}

// ErrUnknownDistro is returned when a distribution ID is not found.
type ErrUnknownDistro struct {
	ID ID
}

func (e *ErrUnknownDistro) Error() string {
	registered := List()
	return fmt.Sprintf("unknown distribution %q, available: %v", e.ID, registered)
}
