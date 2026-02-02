package vendors

import (
	"fmt"
	"sort"
	"sync"
)

// APIConfig holds configuration for a single API connection.
type APIConfig struct {
	Label        string
	Vendor       string
	URL          string
	Credentials  map[string]string
	RateLimit    int
	ResultsLimit int
	CacheTTL     int // Cache TTL in seconds. 0 = never expire (on-demand only), -1 = use default (86400)
}

// APIStatus represents the status of an API connection.
type APIStatus struct {
	Label        string
	Vendor       string
	OrgID        string
	Capabilities []string
	Healthy      bool
	LastError    string
}

// ClientFactory is a function that creates a Client from an APIConfig.
// Vendor implementations register their factory functions at startup.
type ClientFactory func(config *APIConfig) (Client, error)

// APIClientRegistry manages multiple API client instances.
type APIClientRegistry struct {
	clients   map[string]Client     // keyed by API label
	configs   map[string]*APIConfig // original config per label
	factories map[string]ClientFactory
	mu        sync.RWMutex
}

// NewAPIClientRegistry creates a new empty registry.
func NewAPIClientRegistry() *APIClientRegistry {
	return &APIClientRegistry{
		clients:   make(map[string]Client),
		configs:   make(map[string]*APIConfig),
		factories: make(map[string]ClientFactory),
	}
}

// RegisterFactory registers a client factory for a vendor type.
// This should be called during initialization before InitializeClients.
func (r *APIClientRegistry) RegisterFactory(vendor string, factory ClientFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[vendor] = factory
}

// InitializeClients creates clients from the provided configurations.
// Returns a list of initialization errors (non-fatal - partial initialization is allowed).
func (r *APIClientRegistry) InitializeClients(apiConfigs map[string]*APIConfig) []error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var initErrors []error

	for label, config := range apiConfigs {
		r.configs[label] = config

		factory, ok := r.factories[config.Vendor]
		if !ok {
			initErrors = append(initErrors, fmt.Errorf("API %q: unsupported vendor %q", label, config.Vendor))
			continue
		}

		client, err := factory(config)
		if err != nil {
			initErrors = append(initErrors, fmt.Errorf("API %q: %w", label, err))
			continue
		}

		r.clients[label] = client
	}

	return initErrors
}

// GetClient returns the client for a specific API label.
func (r *APIClientRegistry) GetClient(apiLabel string) (Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	client, ok := r.clients[apiLabel]
	if !ok {
		return nil, &APINotFoundError{APILabel: apiLabel}
	}
	return client, nil
}

// MustGetClient returns the client or panics if not found.
//
// Deprecated: MustGetClient panics on error which can crash the application.
// Use GetClient() with proper error handling instead. This function is retained
// only for backward compatibility in tests and should not be used in production code.
func (r *APIClientRegistry) MustGetClient(apiLabel string) Client {
	client, err := r.GetClient(apiLabel)
	if err != nil {
		panic(fmt.Sprintf("MustGetClient(%q): %v - use GetClient() with error handling instead", apiLabel, err))
	}
	return client
}

// GetAllLabels returns all registered API labels in sorted order.
func (r *APIClientRegistry) GetAllLabels() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	labels := make([]string, 0, len(r.clients))
	for label := range r.clients {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	return labels
}

// HasAPI checks if an API label is registered and initialized.
func (r *APIClientRegistry) HasAPI(apiLabel string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.clients[apiLabel]
	return ok
}

// GetVendor returns the vendor type for an API label.
func (r *APIClientRegistry) GetVendor(apiLabel string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	config, ok := r.configs[apiLabel]
	if !ok {
		return "", &APINotFoundError{APILabel: apiLabel}
	}
	return config.Vendor, nil
}

// GetOrgID returns the org ID for an API label.
func (r *APIClientRegistry) GetOrgID(apiLabel string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	config, ok := r.configs[apiLabel]
	if !ok {
		return "", &APINotFoundError{APILabel: apiLabel}
	}
	return config.Credentials["org_id"], nil
}

// GetConfig returns the full config for an API label.
func (r *APIClientRegistry) GetConfig(apiLabel string) (*APIConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	config, ok := r.configs[apiLabel]
	if !ok {
		return nil, &APINotFoundError{APILabel: apiLabel}
	}
	return config, nil
}

// Count returns the number of initialized APIs.
func (r *APIClientRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

// ForEachAPI executes a function for each initialized API sequentially.
// Iteration stops on the first error.
func (r *APIClientRegistry) ForEachAPI(fn func(label string, client Client) error) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get sorted labels for deterministic iteration order
	labels := make([]string, 0, len(r.clients))
	for label := range r.clients {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	for _, label := range labels {
		if err := fn(label, r.clients[label]); err != nil {
			return fmt.Errorf("API %s: %w", label, err)
		}
	}
	return nil
}

// ForEachAPIParallel executes a function for each API in parallel.
// Returns a map of API labels to errors (only contains entries for APIs that had errors).
func (r *APIClientRegistry) ForEachAPIParallel(fn func(label string, client Client) error) map[string]error {
	r.mu.RLock()
	labels := make([]string, 0, len(r.clients))
	clients := make(map[string]Client, len(r.clients))
	for label, client := range r.clients {
		labels = append(labels, label)
		clients[label] = client
	}
	r.mu.RUnlock()

	var wg sync.WaitGroup
	errors := make(map[string]error)
	var mu sync.Mutex

	for _, label := range labels {
		wg.Add(1)
		go func(l string, c Client) {
			defer wg.Done()
			if err := fn(l, c); err != nil {
				mu.Lock()
				errors[l] = err
				mu.Unlock()
			}
		}(label, clients[label])
	}

	wg.Wait()
	return errors
}

// GetStatus returns status for all APIs.
func (r *APIClientRegistry) GetStatus() []APIStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	statuses := make([]APIStatus, 0, len(r.clients))
	for label, client := range r.clients {
		config := r.configs[label]
		statuses = append(statuses, APIStatus{
			Label:        label,
			Vendor:       config.Vendor,
			OrgID:        config.Credentials["org_id"],
			Capabilities: listCapabilities(client),
			Healthy:      true,
		})
	}

	// Sort by label for consistent output
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Label < statuses[j].Label
	})

	return statuses
}

// listCapabilities returns a list of supported capabilities for a client.
func listCapabilities(client Client) []string {
	caps := []string{"sites", "inventory", "devices"} // core capabilities

	if client.Search() != nil {
		caps = append(caps, "search")
	}
	if client.Profiles() != nil {
		caps = append(caps, "profiles")
	}
	if client.Templates() != nil {
		caps = append(caps, "templates")
	}
	if client.Configs() != nil {
		caps = append(caps, "configs")
	}

	return caps
}
