package statter

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/go4org/hashtriemap"
	"github.com/hamba/statter/v2/internal/stats"
)

type registry struct {
	r    Reporter
	hr   *value[HistogramReporter]
	tr   *value[TimingReporter]
	pool *stats.Pool
	cfg  config

	counters   hashtriemap.HashTrieMap[string, *Counter]
	gauges     hashtriemap.HashTrieMap[string, *Gauge]
	histograms hashtriemap.HashTrieMap[string, *Histogram]
	timings    hashtriemap.HashTrieMap[string, *Timing]

	mu       sync.RWMutex
	root     *Statter
	statters map[string]*Statter

	done chan struct{}
	wg   sync.WaitGroup
}

func newRegistry(root *Statter, r Reporter, interval time.Duration, cfg config) *registry {
	reg := &registry{
		r:        r,
		hr:       &value[HistogramReporter]{},
		tr:       &value[TimingReporter]{},
		pool:     stats.NewPool(cfg.percSamples),
		cfg:      cfg,
		root:     root,
		statters: map[string]*Statter{},
		done:     make(chan struct{}),
	}

	if hr, ok := r.(HistogramReporter); ok {
		reg.hr.Store(hr)
	}
	if tr, ok := r.(TimingReporter); ok {
		reg.tr.Store(tr)
	}

	// Register root statter in the deduplication cache.
	k := newKey(root.prefix, root.tags)
	reg.statters[k.SafeString()] = root
	k.Release()

	reg.wg.Add(1)
	go reg.runReportLoop(interval)

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
	r.counters.Range(func(_ string, c *Counter) bool {
		val := c.value()
		if val == 0 {
			return true
		}
		r.r.Counter(c.name, val, c.tags)
		return true
	})

	r.gauges.Range(func(_ string, g *Gauge) bool {
		r.r.Gauge(g.name, g.value(), g.tags)
		return true
	})

	if r.hr.Load() == nil {
		r.histograms.Range(func(_ string, h *Histogram) bool {
			histo := h.value()
			defer r.pool.Put(histo)
			r.reportSample(h.name, "", h.tags, histo)
			return true
		})
	}

	if r.tr.Load() == nil {
		r.timings.Range(func(_ string, t *Timing) bool {
			timing := t.value()
			defer r.pool.Put(timing)
			r.reportSample(t.name, "_ms", t.tags, timing)
			return true
		})
	}
}

func (r *registry) reportSample(name, suffix string, tags [][2]string, sample *stats.Sample) {
	if sample.Count() == 0 {
		return
	}

	prefix := name + "_"
	r.r.Counter(prefix+"count", sample.Count(), tags)
	r.r.Gauge(prefix+"sum"+suffix, sample.Sum(), tags)
	r.r.Gauge(prefix+"mean"+suffix, sample.Mean(), tags)
	r.r.Gauge(prefix+"stddev"+suffix, sample.StdDev(), tags)
	r.r.Gauge(prefix+"min"+suffix, sample.Min(), tags)
	r.r.Gauge(prefix+"max"+suffix, sample.Max(), tags)
	ps := r.cfg.percentiles
	vs := sample.Percentiles(ps)
	for i := range vs {
		n := prefix + strconv.FormatFloat(ps[i], 'g', -1, 64) + "p" + suffix
		r.r.Gauge(n, vs[i], tags)
	}
}

func (r *registry) sampleKeys(name, suffix string) []string {
	prefix := name + "_"
	keys := make([]string, 0, 6+len(r.cfg.percentiles))
	keys = append(keys, prefix+"count")
	keys = append(keys, prefix+"sum"+suffix)
	keys = append(keys, prefix+"mean"+suffix)
	keys = append(keys, prefix+"stddev"+suffix)
	keys = append(keys, prefix+"min"+suffix)
	keys = append(keys, prefix+"max"+suffix)

	for _, p := range r.cfg.percentiles {
		keys = append(keys, prefix+strconv.FormatFloat(p, 'g', -1, 64)+"p"+suffix)
	}

	return keys
}

// SubStatter returns a unique sub statter.
func (r *registry) SubStatter(parent *Statter, prefix string, tags []Tag) *Statter {
	name, newTags := mergeDescriptors(parent.prefix, r.cfg.separator, prefix, parent.tags, tags)

	// Sort merged tags to maintain the sorted-base-tags invariant so that
	// the mergeDescriptors fast paths in metric accessors are safe.
	if len(newTags) > 1 {
		sortTags(newTags)
	}

	k := newKey(name, newTags)
	defer k.Release()

	r.mu.RLock()
	if s, ok := r.statters[k.String()]; ok {
		r.mu.RUnlock()
		return s
	}
	r.mu.RUnlock()

	// Slow path: first time we have seen this sub-statter.
	s := &Statter{
		reg:    r,
		prefix: name,
		tags:   newTags,
	}

	r.mu.Lock()
	if existing, ok := r.statters[k.String()]; ok {
		r.mu.Unlock()
		return existing
	}
	r.statters[k.SafeString()] = s
	r.mu.Unlock()

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
	newTags = mergeTags(newTags, tags)

	return name, newTags
}

func mergeTags(out, tags []Tag) []Tag {
	for _, tag := range tags {
		if i := tagIndex(out, tag[0]); i >= 0 {
			out[i][1] = tag[1]
			continue
		}
		out = append(out, tag)
	}
	return out
}

func tagIndex[T ~[2]string](tags []T, key string) int {
	for i, t := range tags {
		if t[0] == key {
			return i
		}
	}
	return -1
}
