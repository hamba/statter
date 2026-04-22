package statter

import (
	"io"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hamba/statter/v2/internal/stats"
)

// DiscardReporter is a reporter that discards all stats.
var DiscardReporter = discardReporter{}

// Reporter represents a stats reporter.
type Reporter interface {
	Counter(name string, v int64, tags [][2]string)
	Gauge(name string, v float64, tags [][2]string)
}

// RemovableReporter represents a stats reporter that handles removal.
type RemovableReporter interface {
	RemoveCounter(name string, tags [][2]string)
	RemoveGauge(name string, tags [][2]string)
}

// HistogramReporter represents a stats reporter that handles histograms.
type HistogramReporter interface {
	Histogram(name string, tags [][2]string) func(v float64)
}

// RemovableHistogramReporter represents a stats reporter that handles histogram removal.
type RemovableHistogramReporter interface {
	RemoveHistogram(name string, tags [][2]string)
}

// TimingReporter represents a stats reporter that handles timings.
type TimingReporter interface {
	Timing(name string, tags [][2]string) func(v time.Duration)
}

// RemovableTimingReporter represents a stats reporter that handles timing removal.
type RemovableTimingReporter interface {
	RemoveTiming(name string, tags [][2]string)
}

// Tag is a stat tag.
type Tag = [2]string

type config struct {
	prefix      string
	tags        []Tag
	separator   string
	percSamples int
	percentiles []float64
}

func defaultConfig() config {
	return config{
		separator:   ".",
		percSamples: 1000,
		percentiles: []float64{10, 90},
	}
}

// Option represents a statter option function.
type Option func(*config)

// WithPrefix sets the initial prefix on a statter.
func WithPrefix(prefix string) Option {
	return func(c *config) {
		c.prefix = prefix
	}
}

// WithTags sets the initial tags on a statter.
func WithTags(tags ...Tag) Option {
	return func(c *config) {
		c.tags = tags
	}
}

// WithSeparator sets the key separator on a statter.
func WithSeparator(sep string) Option {
	return func(c *config) {
		c.separator = sep
	}
}

// WithPercentileSamples sets the number of samples taken to
// calculate percentiles.
func WithPercentileSamples(n int) Option {
	return func(c *config) {
		c.percSamples = n
	}
}

// WithPercentiles sets the percentiles to calculate and report
// for aggregated histograms and timings.
func WithPercentiles(p []float64) Option {
	return func(c *config) {
		c.percentiles = p
	}
}

// Statter collects and reports stats.
type Statter struct {
	reg    *registry
	prefix string
	tags   []Tag
}

// New returns a statter.
func New(r Reporter, interval time.Duration, opts ...Option) *Statter {
	cfg := defaultConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	// Sort initial tags once to maintain the sorted-base-tags invariant,
	// enabling zero-alloc fast paths in mergeDescriptors.
	if len(cfg.tags) > 1 {
		sortTags(cfg.tags)
	}

	s := &Statter{
		prefix: cfg.prefix,
		tags:   cfg.tags,
	}
	s.reg = newRegistry(s, r, interval, cfg)

	return s
}

// With returns a statter with the given prefix and tags.
func (s *Statter) With(prefix string, tags ...Tag) *Statter {
	return s.reg.SubStatter(s, prefix, tags)
}

// Reporter returns the stats reporter.
//
// The reporter should not be used directly.
func (s *Statter) Reporter() Reporter {
	return s.reg.r
}

// FullName returns the full name with prefix for the given name.
func (s *Statter) FullName(name string) string {
	if s.prefix != "" {
		return s.prefix + s.reg.cfg.separator + name
	}
	return name
}

// HasCounter determines if the counter exists.
func (s *Statter) HasCounter(name string, tags ...Tag) bool {
	n, t := s.mergeDescriptors(name, tags)
	k := newKey(n, t)

	_, ok := s.reg.counters.Load(k.String())

	putKey(k)
	return ok
}

// Counter returns a counter for the given name and tags.
func (s *Statter) Counter(name string, tags ...Tag) *Counter {
	n, t := s.mergeDescriptors(name, tags)
	k := newKey(n, t)

	c, ok := s.reg.counters.Load(k.String())
	if !ok {
		counter := &Counter{
			name: n,
			tags: t,
			key:  k.String(),
			reg:  s.reg,
		}
		c, _ = s.reg.counters.LoadOrStore(k.String(), counter)
	}

	putKey(k)

	return c
}

// HasGauge determines if the gauge exists.
func (s *Statter) HasGauge(name string, tags ...Tag) bool {
	n, t := s.mergeDescriptors(name, tags)
	k := newKey(n, t)

	_, ok := s.reg.gauges.Load(k.String())

	putKey(k)

	return ok
}

// Gauge returns a gauge for the given name and tags.
func (s *Statter) Gauge(name string, tags ...Tag) *Gauge {
	n, t := s.mergeDescriptors(name, tags)
	k := newKey(n, t)

	g, ok := s.reg.gauges.Load(k.String())
	if !ok {
		gauge := &Gauge{
			name: n,
			tags: t,
			key:  k.String(),
			reg:  s.reg,
		}
		g, _ = s.reg.gauges.LoadOrStore(k.String(), gauge)
	}

	putKey(k)

	return g
}

// HasHistogram determines if the histogram exists.
func (s *Statter) HasHistogram(name string, tags ...Tag) bool {
	n, t := s.mergeDescriptors(name, tags)
	k := newKey(n, t)

	_, ok := s.reg.histograms.Load(k.String())

	putKey(k)

	return ok
}

// Histogram returns a histogram for the given name and tags.
func (s *Statter) Histogram(name string, tags ...Tag) *Histogram {
	n, t := s.mergeDescriptors(name, tags)
	k := newKey(n, t)

	h, ok := s.reg.histograms.Load(k.String())
	if !ok {
		histogram := newHistogram(s.reg.hr.Load(), n, t, s.reg.pool)
		histogram.key = k.SafeString()
		histogram.reg = s.reg
		h, _ = s.reg.histograms.LoadOrStore(k.String(), histogram)
	}

	putKey(k)

	return h
}

// HasTiming determines if the timing exists.
func (s *Statter) HasTiming(name string, tags ...Tag) bool {
	n, t := s.mergeDescriptors(name, tags)
	k := newKey(n, t)

	_, ok := s.reg.timings.Load(k.String())

	putKey(k)

	return ok
}

// Timing returns a timing for the given name and tags.
func (s *Statter) Timing(name string, tags ...Tag) *Timing {
	n, tags := s.mergeDescriptors(name, tags)
	k := newKey(n, tags)

	t, ok := s.reg.timings.Load(k.String())
	if !ok {
		timing := newTiming(s.reg.tr.Load(), n, tags, s.reg.pool)
		timing.key = k.SafeString()
		timing.reg = s.reg
		t, _ = s.reg.timings.LoadOrStore(k.String(), timing)
	}

	putKey(k)

	return t
}

func (s *Statter) mergeDescriptors(name string, tags []Tag) (string, []Tag) {
	if s.prefix != "" {
		name = s.prefix + s.reg.cfg.separator + name
	}

	// Fast path: no tags at all.
	if len(s.tags) == 0 && len(tags) == 0 {
		return name, nil
	}

	// Fast path: no base tags, single call-site tag — no key duplication possible.
	// The variadic is a fresh slice per call; safe to return directly.
	if len(s.tags) == 0 && len(tags) == 1 {
		return name, tags
	}

	// Fast path: no call-site tags — return base tags directly.
	// Safe because s.tags is pre-sorted (invariant) and immutable after statter
	// creation. newKey calls sortTags which is a no-op read for sorted input.
	if len(tags) == 0 {
		return name, s.tags
	}

	// General path: merge base tags and call-site tags with deduplication.
	// Also handles multiple call-site tags that may share the same key.
	newTags := make([]Tag, len(s.tags), len(s.tags)+len(tags))
	copy(newTags, s.tags)
	for _, tag := range tags {
		if i := tagIndex(newTags, tag[0]); i >= 0 {
			newTags[i][1] = tag[1]
		} else {
			newTags = append(newTags, tag)
		}
	}

	return name, newTags
}

func tagIndex[T ~[2]string](tags []T, key string) int {
	for i, t := range tags {
		if t[0] == key {
			return i
		}
	}
	return -1
}

// Close closes the statter and reporter.
func (s *Statter) Close() error {
	if err := s.reg.Close(s); err != nil {
		return err
	}

	if c, ok := s.reg.r.(io.Closer); ok {
		if err := c.Close(); err != nil {
			return err
		}
	}

	return nil
}

// Counter implements a counter.
type Counter struct {
	name string
	tags [][2]string
	key  string
	reg  *registry

	val atomic.Int64
}

// Inc increments the counter.
func (c *Counter) Inc(v int64) {
	c.val.Add(v)
}

// Delete removes the counter.
func (c *Counter) Delete() {
	if rr, ok := c.reg.r.(RemovableReporter); ok {
		rr.RemoveCounter(c.name, c.tags)
	}
	_, _ = c.reg.counters.LoadAndDelete(c.key)
}

func (c *Counter) value() int64 {
	return c.val.Swap(0)
}

// Gauge implements a gauge.
type Gauge struct {
	name string
	tags [][2]string
	key  string
	reg  *registry

	val atomic.Uint64
}

// Set sets the gauge value.
func (g *Gauge) Set(v float64) {
	g.val.Store(math.Float64bits(v))
}

// Inc increments the gauge by 1.
func (g *Gauge) Inc() {
	g.Add(1)
}

// Dec decrements the gauge by 1.
func (g *Gauge) Dec() {
	g.Add(-1)
}

// Add increases the gauge's value by the argument.
// The operation is thread-safe.
func (g *Gauge) Add(v float64) {
	for {
		oldBits := g.val.Load()
		newBits := math.Float64bits(math.Float64frombits(oldBits) + v)
		if g.val.CompareAndSwap(oldBits, newBits) {
			return
		}
	}
}

// Sub subtracts the argument from the gauge's value.
func (g *Gauge) Sub(v float64) {
	g.Add(v * -1)
}

// Delete remove the gauge.
func (g *Gauge) Delete() {
	if rr, ok := g.reg.r.(RemovableReporter); ok {
		rr.RemoveGauge(g.name, g.tags)
	}
	_, _ = g.reg.gauges.LoadAndDelete(g.key)
}

func (g *Gauge) value() float64 {
	v := g.val.Load()
	return math.Float64frombits(v)
}

// Histogram implements a histogram.
type Histogram struct {
	hrFn func(v float64)
	name string
	tags [][2]string
	key  string
	reg  *registry
	pool *stats.Pool

	mu sync.Mutex
	s  *stats.Sample
}

func newHistogram(hr HistogramReporter, name string, tags [][2]string, pool *stats.Pool) *Histogram {
	if hr != nil {
		fn := hr.Histogram(name, tags)
		if fn != nil {
			return &Histogram{
				hrFn: fn,
				name: name,
				tags: tags,
			}
		}
	}

	return &Histogram{
		name: name,
		tags: tags,
		pool: pool,
		s:    pool.Get(),
	}
}

// Observe observes a histogram value.
func (h *Histogram) Observe(v float64) {
	if h.hrFn != nil {
		h.hrFn(v)
		return
	}

	h.mu.Lock()
	h.s.Add(v)
	h.mu.Unlock()
}

// Delete removes the histogram.
func (h *Histogram) Delete() {
	if rtr, ok := h.reg.r.(RemovableHistogramReporter); ok {
		rtr.RemoveHistogram(h.name, h.tags)
	} else if rr, ok := h.reg.r.(RemovableReporter); ok {
		for _, k := range h.reg.sampleKeys(h.name, "") {
			rr.RemoveGauge(k, h.tags)
		}
	}
	_, _ = h.reg.histograms.LoadAndDelete(h.key)
}

func (h *Histogram) value() *stats.Sample {
	h.mu.Lock()
	s := h.s
	h.s = h.pool.Get()
	h.mu.Unlock()

	return s
}

// Timing implements a timing.
type Timing struct {
	trFn func(v time.Duration)
	name string
	tags [][2]string
	key  string
	reg  *registry
	pool *stats.Pool

	mu sync.Mutex
	s  *stats.Sample
}

func newTiming(tr TimingReporter, name string, tags [][2]string, pool *stats.Pool) *Timing {
	if tr != nil {
		fn := tr.Timing(name, tags)
		if fn != nil {
			return &Timing{
				trFn: fn,
				name: name,
				tags: tags,
			}
		}
	}

	return &Timing{
		name: name,
		tags: tags,
		pool: pool,
		s:    pool.Get(),
	}
}

// Observe observes a timing duration.
//
// If the reporter does not handle timings, the duration
// will be aggregated in milliseconds.
func (t *Timing) Observe(d time.Duration) {
	if t.trFn != nil {
		t.trFn(d)
		return
	}

	t.mu.Lock()
	t.s.Add(d.Seconds() * 1000)
	t.mu.Unlock()
}

// Delete removes the timing.
func (t *Timing) Delete() {
	if rtr, ok := t.reg.r.(RemovableTimingReporter); ok {
		rtr.RemoveTiming(t.name, t.tags)
	} else if rr, ok := t.reg.r.(RemovableReporter); ok {
		for _, k := range t.reg.sampleKeys(t.name, "_ms") {
			rr.RemoveGauge(k, t.tags)
		}
	}
	_, _ = t.reg.timings.LoadAndDelete(t.key)
}

func (t *Timing) value() *stats.Sample {
	t.mu.Lock()
	s := t.s
	t.s = t.pool.Get()
	t.mu.Unlock()

	return s
}

type discardReporter struct{}

func (r discardReporter) Counter(_ string, _ int64, _ [][2]string) {}
func (r discardReporter) Gauge(_ string, _ float64, _ [][2]string) {}

func (r discardReporter) Histogram(_ string, _ [][2]string) func(float64) {
	return func(_ float64) {}
}

func (r discardReporter) Timing(_ string, _ [][2]string) func(time.Duration) {
	return func(_ time.Duration) {}
}

type value[T any] struct {
	val atomic.Value
}

func (v *value[T]) Load() T {
	var zeroT T
	val, ok := v.val.Load().(T)
	if !ok {
		return zeroT
	}
	return val
}

func (v *value[T]) Store(t T) {
	v.val.Store(t)
}
