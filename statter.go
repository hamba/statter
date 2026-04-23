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

// New returns a Statter that aggregates stats and flushes them to r on every
// interval tick. Options may be used to set an initial prefix, tags, key
// separator, and percentile configuration.
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

// With returns a sub-statter whose metrics are prefixed with prefix and carry
// the additional tags. The prefix is joined to the parent prefix with the
// configured separator. Tags are merged with the parent tags; a call-site tag
// whose key already exists in the parent overrides the parent value.
//
// Sub-statters with the same resolved prefix and tags are deduplicated: the
// same instance is returned for repeated calls with identical arguments.
func (s *Statter) With(prefix string, tags ...Tag) *Statter {
	return s.reg.SubStatter(s, prefix, tags)
}

// Reporter returns the underlying stats reporter.
//
// The reporter is exposed for advanced use cases such as pre-registering
// metrics with helpers like RegisterCounter, RegisterGauge, and
// RegisterHistogram. Observing or mutating stats through the reporter
// directly bypasses aggregation and should otherwise be avoided.
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
	k := s.key(name, tags)

	_, ok := s.reg.counters.Load(k.String())

	k.Release()

	return ok
}

// Counter returns a counter for the given name and tags. The counter is
// created on the first call and the same instance is returned for subsequent
// calls with identical name and tags.
func (s *Statter) Counter(name string, tags ...Tag) *Counter {
	k := s.key(name, tags)

	c, ok := s.reg.counters.Load(k.String())
	if !ok {
		n, t := s.mergeDescriptors(name, tags)
		counter := &Counter{
			name: n,
			tags: t,
			key:  k.SafeString(),
			reg:  s.reg,
		}
		c, _ = s.reg.counters.LoadOrStore(k.SafeString(), counter)
	}

	k.Release()

	return c
}

// HasGauge determines if the gauge exists.
func (s *Statter) HasGauge(name string, tags ...Tag) bool {
	k := s.key(name, tags)

	_, ok := s.reg.gauges.Load(k.String())

	k.Release()

	return ok
}

// Gauge returns a gauge for the given name and tags. The gauge is created on
// the first call and the same instance is returned for subsequent calls with
// identical name and tags.
func (s *Statter) Gauge(name string, tags ...Tag) *Gauge {
	k := s.key(name, tags)

	g, ok := s.reg.gauges.Load(k.String())
	if !ok {
		n, t := s.mergeDescriptors(name, tags)
		gauge := &Gauge{
			name: n,
			tags: t,
			key:  k.SafeString(),
			reg:  s.reg,
		}
		g, _ = s.reg.gauges.LoadOrStore(k.SafeString(), gauge)
	}

	k.Release()

	return g
}

// HasHistogram determines if the histogram exists.
func (s *Statter) HasHistogram(name string, tags ...Tag) bool {
	k := s.key(name, tags)

	_, ok := s.reg.histograms.Load(k.String())

	k.Release()

	return ok
}

// Histogram returns a histogram for the given name and tags. The histogram is
// created on the first call and the same instance is returned for subsequent
// calls with identical name and tags.
//
// When the reporter implements [HistogramReporter], observations are delegated
// to it directly. Otherwise observations are aggregated locally and reported
// each interval as a set of gauges (_sum, _mean, _stddev, _min, _max, and
// each configured percentile) plus a _count counter.
func (s *Statter) Histogram(name string, tags ...Tag) *Histogram {
	k := s.key(name, tags)

	h, ok := s.reg.histograms.Load(k.String())
	if !ok {
		n, t := s.mergeDescriptors(name, tags)
		histogram := newHistogram(s.reg.hr.Load(), n, t, s.reg.pool)
		histogram.key = k.SafeString()
		histogram.reg = s.reg
		h, _ = s.reg.histograms.LoadOrStore(k.SafeString(), histogram)
	}

	k.Release()

	return h
}

// HasTiming determines if the timing exists.
func (s *Statter) HasTiming(name string, tags ...Tag) bool {
	k := s.key(name, tags)

	_, ok := s.reg.timings.Load(k.String())

	k.Release()

	return ok
}

// Timing returns a timing for the given name and tags. The timing is created
// on the first call and the same instance is returned for subsequent calls
// with identical name and tags.
//
// When the reporter implements [TimingReporter], observations are delegated
// to it directly. Otherwise observations are aggregated locally in
// milliseconds and reported each interval as a set of gauges
// (_sum_ms, _mean_ms, _stddev_ms, _min_ms, _max_ms, and each configured
// percentile) plus a _count counter.
func (s *Statter) Timing(name string, tags ...Tag) *Timing {
	k := s.key(name, tags)

	t, ok := s.reg.timings.Load(k.String())
	if !ok {
		n, tags := s.mergeDescriptors(name, tags)
		timing := newTiming(s.reg.tr.Load(), n, tags, s.reg.pool)
		timing.key = k.SafeString()
		timing.reg = s.reg
		t, _ = s.reg.timings.LoadOrStore(k.SafeString(), timing)
	}

	k.Release()

	return t
}

func (s *Statter) key(name string, tags []Tag) *key {
	switch {
	case s.prefix != "" && name != "":
		name = s.prefix + s.reg.cfg.separator + name
	case name == "":
		name = s.prefix
	}

	keyTags := make([]Tag, len(s.tags), len(s.tags)+len(tags))
	copy(keyTags, s.tags)
	keyTags = mergeTags(keyTags, tags)

	return newKey(name, keyTags)
}

func (s *Statter) mergeDescriptors(name string, tags []Tag) (string, []Tag) {
	return mergeDescriptors(s.prefix, s.reg.cfg.separator, name, s.tags, tags)
}

// Close stops the reporting loop, flushes any pending stats to the reporter,
// and closes the reporter if it implements [io.Closer]. Close must be called
// on the root statter; calling it on a sub-statter returns an error.
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

// Counter implements a counter that monotonically accumulates a value between
// flushes. The accumulated delta is reported and reset to zero on each flush.
type Counter struct {
	name string
	tags [][2]string
	key  string
	reg  *registry

	val atomic.Int64
}

// Inc increments the counter by v.
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

// Gauge implements a gauge that holds its last-set value and reports it on
// each flush.
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

// Add increases the gauge's value by v.
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

// Delete removes the gauge.
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
//
// When the reporter implements [HistogramReporter], observations are delegated
// to it directly. Otherwise observations are aggregated locally and reported
// each interval as a set of gauges (_sum, _mean, _stddev, _min, _max, and
// each configured percentile) plus a _count counter.
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
//
// When the reporter implements [TimingReporter], observations are delegated to
// it directly. Otherwise observations are aggregated locally in milliseconds
// and reported each interval as a set of gauges (_sum_ms, _mean_ms,
// _stddev_ms, _min_ms, _max_ms, and each configured percentile) plus a
// _count counter.
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
