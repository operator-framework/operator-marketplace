package datastore

import (
	"fmt"
)

func newUpdateResult(opSrc string) *UpdateResult {
	return &UpdateResult{
		opSrc:   opSrc,
		Updated: make([]string, 0),
		Removed: make([]string, 0),
	}
}

func NewPackageUpdateAggregator(opSrc string) *PackageUpdateAggregator {
	return &PackageUpdateAggregator{
		opSrc:   opSrc,
		updated: map[string]bool{},
		removed: map[string]bool{},
	}
}

// NewPackageRefreshNotification is used to update packages associated with
// the OperatorSource.
func NewPackageRefreshNotification(opSrc string) *PackageUpdateAggregator {
	return &PackageUpdateAggregator{
		opSrc:         opSrc,
		refreshNeeded: true,
	}
}

// UpdateResult holds information related to what has changed in the remote
// registry associated with an operator source.
type UpdateResult struct {
	opSrc string
	// RegistryHasUpdate indicates whether the remote registry associated with
	// the operatour source has any change. It is set to true if any of the
	// following is true:
	// a. A new repository has been pushed.
	// b. An existing repository has been removed.
	// c. An existing repository has a new version.
	RegistryHasUpdate bool

	// Updated is the list of operator name(s) that potentially have new
	// version(s) because the corresponding repositories have new version(s).
	Updated []string

	// Removed is the list of operator name(s) that are no longer available
	// because the corresponding repositories have been removed.
	Removed []string
}

func (a *UpdateResult) String() string {
	return fmt.Sprintf("operator(s) updated=%s, operator(s) removed=%s", a.Updated, a.Removed)
}

// PackageUpdateNotification is an interface used to determine whether a
// specified operator has a new version or has been removed.
type PackageUpdateNotification interface {
	// GetOpSrc returns the name of the OperaterSource that update should
	// be applied to.
	GetOpSrc() string

	// IsRemoved returns true if the specified package has been removed.
	IsRemoved(pkg string) bool

	// IsUpdated returns true if the specified package has a new version.
	IsUpdated(pkg string) bool

	// IsRefreshNotification returns true if the notification is used to update the
	// initial state. We use this on startup and as a way to force update when
	// the cache is in a bad state.
	IsRefreshNotification() bool
}

// PackageUpdateAggregator is used to aggregate update information from across
// all operator source(s).
// PackageUpdateAggregator also implements PackageUpdateNotification interface.
type PackageUpdateAggregator struct {
	opSrc         string
	updated       map[string]bool
	removed       map[string]bool
	refreshNeeded bool
}

func (a *PackageUpdateAggregator) IsRefreshNotification() bool {
	return a.refreshNeeded
}

func (a *PackageUpdateAggregator) GetOpSrc() string {
	return a.opSrc
}

// Add accepts an UpdateResult for a given operator source and aggregates it.
func (a *PackageUpdateAggregator) Add(result *UpdateResult) {
	for _, pkg := range result.Updated {
		a.updated[pkg] = true
	}

	for _, pkg := range result.Removed {
		a.removed[pkg] = true
	}
}

// IsUpdatedOrRemoved returns true whether any operator has a new version or has
// been removed from the remote registry.
func (a *PackageUpdateAggregator) IsUpdatedOrRemoved() bool {
	return len(a.removed) > 0 || len(a.updated) > 0
}

func (a *PackageUpdateAggregator) String() string {
	ulist := make([]string, 0)
	for k, _ := range a.updated {
		ulist = append(ulist, k)
	}

	rlist := make([]string, 0)
	for k, _ := range a.removed {
		rlist = append(rlist, k)
	}

	return fmt.Sprintf("operator(s) updated=%s, operator(s) removed=%s", ulist, rlist)
}

func (a *PackageUpdateAggregator) IsRemoved(pkg string) bool {
	_, exists := a.removed[pkg]
	return exists
}

func (a *PackageUpdateAggregator) IsUpdated(pkg string) bool {
	_, exists := a.updated[pkg]
	return exists
}
