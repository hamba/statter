package statter

import (
	"errors"
	"sync"
	"time"

	"github.com/puzpuzpuz/xsync"
)

type registry struct {
	mu       sync.Mutex
	root     *Statter
	statters map[string]*Statter

	done chan struct{}
	wg   sync.WaitGroup
}

func newRegistry(root *Statter, d time.Duration) *registry {
	name, tags := mergeDescriptors("", root.cfg.separator, root.prefix, nil, root.tags)
	k := newKey(name, tags)
	defer putKey(k)

	reg := &registry{
		root: root,
		statters: map[string]*Statter{
			k.SafeString(): root,
		},
		done: make(chan struct{}),
	}

	reg.wg.Add(1)
	go reg.runReportLoop(d)

	return reg
}

func (r *registry) runReportLoop(d time.Duration) {
	defer r.wg.Done()

	tick := time.NewTicker(d)
	defer tick.Stop()

	for {
		select {
		case <-r.done:
			return
		case <-tick.C:
		}

		r.report()
	}
}

func (r *registry) report() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, s := range r.statters {
		s.report()
	}
}

// SubStatter returns a unique sub statter.
func (r *registry) SubStatter(parent *Statter, prefix string, tags []Tag) *Statter {
	name, tags := mergeDescriptors(parent.prefix, parent.cfg.separator, prefix, parent.tags, tags)

	k := newKey(name, tags)
	defer putKey(k)

	r.mu.Lock()
	defer r.mu.Unlock()

	if s, ok := r.statters[k.String()]; ok {
		return s
	}

	s := &Statter{
		cfg:        parent.cfg,
		reg:        r,
		r:          parent.r,
		hr:         parent.hr,
		tr:         parent.tr,
		pool:       parent.pool,
		prefix:     name,
		tags:       tags,
		counters:   xsync.NewMapOf[*Counter](),
		gauges:     xsync.NewMapOf[*Gauge](),
		histograms: xsync.NewMapOf[*Histogram](),
		timings:    xsync.NewMapOf[*Timing](),
	}
	r.statters[k.SafeString()] = s

	return s
}

// Close closes the registry if the caller is the root statter,
// otherwise an error is returned.
func (r *registry) Close(caller *Statter) error {
	if caller != r.root {
		return errors.New("close cannot be called from a sub-statter")
	}

	close(r.done)
	r.wg.Wait()

	r.report()

	return nil
}

func mergeDescriptors(prefix, sep, name string, baseTags, tags []Tag) (string, []Tag) {
	switch {
	case prefix != "" && name != "":
		name = prefix + sep + name
	case name == "":
		name = prefix
	}

	newTags := make([]Tag, 0, len(baseTags)+len(tags))
	newTags = append(newTags, baseTags...)
	newTags = append(newTags, tags...)

	return name, newTags
}
