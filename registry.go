package statter

import (
	"errors"
	"sync"
	"time"
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
	snapshot := make([]*Statter, 0, len(r.statters))
	for _, s := range r.statters {
		snapshot = append(snapshot, s)
	}
	r.mu.Unlock()

	for _, s := range snapshot {
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
		cfg:    parent.cfg,
		reg:    r,
		r:      parent.r,
		hr:     parent.hr,
		tr:     parent.tr,
		pool:   parent.pool,
		prefix: name,
		tags:   tags,
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

	newTags := make([]Tag, len(baseTags), len(baseTags)+len(tags))
	copy(newTags, baseTags)
	for _, tag := range tags {
		if i := tagIndex(newTags, tag[0]); i >= 0 {
			newTags[i][1] = tag[1]
			continue
		}
		newTags = append(newTags, tag)
	}

	return name, newTags
}
