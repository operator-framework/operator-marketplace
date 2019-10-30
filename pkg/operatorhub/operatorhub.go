package operatorhub

import (
	"sync"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/operator-framework/operator-marketplace/pkg/defaults"
)

// DefaultName is the default name of the OperatorHub config resource on an
// OpenShift cluster
const DefaultName = "cluster"

// instance is the singleton hubconfig object
var instance *operatorhub

func init() {
	instance = &operatorhub{
		current: make(map[string]bool),
		lock:    sync.Mutex{},
	}
}

// operatorhub implements OperatorHub
type operatorhub struct {
	current map[string]bool
	lock    sync.Mutex
}

// OperatorHub is the interface to interact with the OperatorHub configuration in
//  memory.
type OperatorHub interface {
	Get() map[string]bool
	Set(spec configv1.OperatorHubSpec)
	Disabled() bool
}

// GetSingleton returns the singleton instance of HubConfig
func GetSingleton() OperatorHub {
	return instance
}

// Get returns the current configuration
func (o *operatorhub) Get() map[string]bool {
	o.lock.Lock()
	defer o.lock.Unlock()
	return o.current
}

// Disabled returns true if all defaults are disabled
func (o *operatorhub) Disabled() bool {
	o.lock.Lock()
	defer o.lock.Unlock()

	for _, disabled := range o.current {
		if disabled == false {
			return false
		}
	}
	return true
}

// Set sets the current configuration based on the spec. If the spec is empty,
// then the defaults are set. If spec.DisableAllDefaultSources is true, then
// all defaults are marked as disabled. However if sources contains a source
// that is marked as not disabled, then that take precedence.
func (o *operatorhub) Set(spec configv1.OperatorHubSpec) {
	o.lock.Lock()
	defer o.lock.Unlock()

	// Reset to the defaults. If DisableAllDefaultSources, mark all defaults
	// as disabled.
	o.current = make(map[string]bool)
	for k, v := range defaults.GetDefaultConfig() {
		if spec.DisableAllDefaultSources {
			o.current[k] = true
		} else {
			o.current[k] = v
		}
	}

	// Override with what is in the spec.Sources
	for _, source := range spec.Sources {
		o.current[source.Name] = source.Disabled
	}
}
