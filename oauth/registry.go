package oauth

import (
	"fmt"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

var (
	providers = make(map[string]Provider)
	mu        sync.RWMutex
	// customProviderSlugs tracks which providers are custom (can be unregistered)
	customProviderSlugs = make(map[string]bool)
)

// Register registers an OAuth provider with the given name
func Register(name string, provider Provider) {
	mu.Lock()
	defer mu.Unlock()
	providers[name] = provider
}

// RegisterCustom registers a custom OAuth provider without replacing a built-in provider.
func RegisterCustom(name string, provider Provider) {
	registerCustom(name, provider)
}

func registerCustom(name string, provider Provider) bool {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := providers[name]; exists && !customProviderSlugs[name] {
		return false
	}
	providers[name] = provider
	customProviderSlugs[name] = true
	return true
}

// Unregister removes a provider from the registry
func Unregister(name string) {
	mu.Lock()
	defer mu.Unlock()
	delete(providers, name)
	delete(customProviderSlugs, name)
}

// GetProvider returns the OAuth provider for the given name
func GetProvider(name string) Provider {
	mu.RLock()
	defer mu.RUnlock()
	return providers[name]
}

// GetAllProviders returns all registered OAuth providers
func GetAllProviders() map[string]Provider {
	mu.RLock()
	defer mu.RUnlock()
	result := make(map[string]Provider, len(providers))
	for k, v := range providers {
		result[k] = v
	}
	return result
}

// GetEnabledCustomProviders returns all enabled custom OAuth providers
func GetEnabledCustomProviders() []*GenericOAuthProvider {
	mu.RLock()
	defer mu.RUnlock()
	var result []*GenericOAuthProvider
	for name, provider := range providers {
		if customProviderSlugs[name] {
			if gp, ok := provider.(*GenericOAuthProvider); ok && gp.IsEnabled() {
				result = append(result, gp)
			}
		}
	}
	return result
}

// IsProviderRegistered checks if a provider is registered
func IsProviderRegistered(name string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := providers[name]
	return ok
}

// IsCustomProvider checks if a provider is a custom provider
func IsCustomProvider(name string) bool {
	mu.RLock()
	defer mu.RUnlock()
	return customProviderSlugs[name]
}

// LoadCustomProviders loads all custom OAuth providers from the database
func LoadCustomProviders() error {
	// First, unregister all existing custom providers
	mu.Lock()
	for name := range customProviderSlugs {
		delete(providers, name)
	}
	customProviderSlugs = make(map[string]bool)
	mu.Unlock()

	// Load all custom providers from database
	customProviders, err := model.GetAllCustomOAuthProviders()
	if err != nil {
		common.SysError("Failed to load custom OAuth providers: " + err.Error())
		return err
	}

	// Register each custom provider
	loaded := 0
	for _, config := range customProviders {
		provider := NewGenericOAuthProvider(config)
		if !registerCustom(config.Slug, provider) {
			common.SysError("Skipped custom OAuth provider with reserved built-in slug: " + config.Slug)
			continue
		}
		loaded++
		common.SysLog("Loaded custom OAuth provider: " + config.Name + " (" + config.Slug + ")")
	}

	common.SysLog(fmt.Sprintf("Loaded %d custom OAuth providers", loaded))
	return nil
}

// ReloadCustomProviders reloads all custom OAuth providers from the database
func ReloadCustomProviders() error {
	return LoadCustomProviders()
}

// RegisterOrUpdateCustomProvider registers or updates a single custom provider
func RegisterOrUpdateCustomProvider(config *model.CustomOAuthProvider) {
	provider := NewGenericOAuthProvider(config)
	mu.Lock()
	defer mu.Unlock()
	if _, exists := providers[config.Slug]; exists && !customProviderSlugs[config.Slug] {
		common.SysError("Skipped custom OAuth provider with reserved built-in slug: " + config.Slug)
		return
	}
	providers[config.Slug] = provider
	customProviderSlugs[config.Slug] = true
}

// UnregisterCustomProvider unregisters a custom provider by slug
func UnregisterCustomProvider(slug string) {
	mu.Lock()
	defer mu.Unlock()
	if !customProviderSlugs[slug] {
		return
	}
	delete(providers, slug)
	delete(customProviderSlugs, slug)
}
