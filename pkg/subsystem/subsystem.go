// Package subsystem defines the extensibility seam that lets create-list
// support multiple subsystems, each discovering its own component images.
package subsystem

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/spf13/pflag"

	"github.com/bk201/image-tool/pkg/fetch"
)

// ComponentResult is the set of images discovered for a single component of
// a subsystem.
type ComponentResult struct {
	Component string
	Images    []string
}

// Options carries the inputs a Subsystem needs to discover its images.
type Options struct {
	Fetcher fetch.Fetcher
	Version string
}

// Subsystem discovers the container images that make up one version of a
// piece of software, split out by component.
type Subsystem interface {
	Name() string
	Collect(ctx context.Context, o Options) ([]ComponentResult, error)
}

// FlagRegistrar is implemented by subsystems that expose their own CLI
// flags. Flags must be named "--<subsystem>-<flag>" since cobra parses all
// flags before the subsystem argument is known, so every registered
// subsystem's flags are always present.
type FlagRegistrar interface {
	RegisterFlags(fs *pflag.FlagSet)
}

// Verifier is implemented by subsystems that can cross-check discovered
// images against an authoritative upstream list.
type Verifier interface {
	// Verify returns the subset of images that are missing from the
	// upstream list. A non-nil error means the check itself could not be
	// performed (e.g. the upstream list could not be fetched); callers
	// should treat that as a warning, not a fatal condition.
	Verify(ctx context.Context, o Options, images []string) (missing []string, err error)
}

var (
	mu       sync.RWMutex
	registry = map[string]Subsystem{}
)

// Register adds s to the registry. It panics if a subsystem with the same
// name is already registered, since that indicates a programming error.
func Register(s Subsystem) {
	mu.Lock()
	defer mu.Unlock()

	name := s.Name()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("subsystem: duplicate registration for %q", name))
	}
	registry[name] = s
}

// Get looks up a registered subsystem by name.
func Get(name string) (Subsystem, bool) {
	mu.RLock()
	defer mu.RUnlock()

	s, ok := registry[name]
	return s, ok
}

// Names returns the names of all registered subsystems, sorted.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
