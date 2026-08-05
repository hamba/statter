package statter

import (
	"sync"
	"sync/atomic"
)

// deletable is a metric that can remove itself from the registry and the
// reporter.
type deletable interface {
	Delete()
}

// Scope is a sub-statter identified by a unique id.
//
// Metrics created through a Scope are tracked by it, and are removed together
// when the Scope is deleted or replaced. See [Statter.Scope].
type Scope struct {
	reg    *registry
	id     string
	parent *Statter
	tags   []Tag
	s      *Statter

	deleted atomic.Bool

	mu      sync.Mutex
	metrics []deletable
}

func newScope(reg *registry, parent *Statter, id string, tags []Tag) *Scope {
	// The tags are owned by the caller, so they must be copied before they are
	// merged and sorted.
	rawTags := make([]Tag, len(tags))
	copy(rawTags, tags)

	name, scopeTags := mergeDescriptors(parent.prefix, reg.cfg.separator, "", parent.tags, rawTags)

	return &Scope{
		reg:    reg,
		id:     id,
		parent: parent,
		tags:   rawTags,
		// The scope statter is intentionally not registered with the registry
		// statter cache, as that cache is never evicted.
		s: &Statter{
			reg:    reg,
			prefix: name,
			tags:   scopeTags,
		},
	}
}

// matches determines if the scope was created from the given parent and tags.
//
// The tags are compared as given, rather than as merged, as identical input on
// the same parent always resolves to the same scope. This keeps the comparison
// allocation free.
func (sc *Scope) matches(parent *Statter, tags []Tag) bool {
	if sc.parent != parent || len(sc.tags) != len(tags) {
		return false
	}
	for i, tag := range tags {
		if sc.tags[i] != tag {
			return false
		}
	}
	return true
}

// FullName returns the full name with prefix for the given name.
func (sc *Scope) FullName(name string) string {
	return sc.s.FullName(name)
}

// Counter returns a counter for the given name and tags, tracking it against
// the scope.
func (sc *Scope) Counter(name string, tags ...Tag) *Counter {
	c, created := sc.s.counter(name, tags)
	if created {
		sc.track(c)
	}
	return c
}

// Gauge returns a gauge for the given name and tags, tracking it against the
// scope.
func (sc *Scope) Gauge(name string, tags ...Tag) *Gauge {
	g, created := sc.s.gauge(name, tags)
	if created {
		sc.track(g)
	}
	return g
}

// Histogram returns a histogram for the given name and tags, tracking it
// against the scope.
func (sc *Scope) Histogram(name string, tags ...Tag) *Histogram {
	h, created := sc.s.histogram(name, tags)
	if created {
		sc.track(h)
	}
	return h
}

// Timing returns a timing for the given name and tags, tracking it against the
// scope.
func (sc *Scope) Timing(name string, tags ...Tag) *Timing {
	t, created := sc.s.timing(name, tags)
	if created {
		sc.track(t)
	}
	return t
}

func (sc *Scope) track(m deletable) {
	sc.mu.Lock()
	if sc.deleted.Load() {
		sc.mu.Unlock()
		m.Delete()
		return
	}
	sc.metrics = append(sc.metrics, m)
	sc.mu.Unlock()
}

// Delete removes the scope and all metrics created through it.
//
// Delete is idempotent. Metrics requested from a deleted scope are returned to
// the caller but are not tracked, and are removed immediately.
func (sc *Scope) Delete() {
	sc.delete(true)
}

func (sc *Scope) delete(unregister bool) {
	if !sc.deleted.CompareAndSwap(false, true) {
		return
	}

	if unregister {
		_ = sc.reg.scopes.CompareAndDelete(sc.id, sc)
	}

	sc.mu.Lock()
	metrics := sc.metrics
	sc.metrics = nil
	sc.mu.Unlock()

	// The metrics take the registry report lock themselves, so it must not be
	// held here. A sync.RWMutex is not reentrant.
	for _, m := range metrics {
		m.Delete()
	}
}
