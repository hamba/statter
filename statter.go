package statter

import (
	"io"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hamba/statter/v2/internal/stats"
)

// DiscardReporter discards all reported metrics.
var DiscardReporter = discardReporter{}

// Reporter receives counter deltas and gauge values from a Statter.
// The statter passes tags sorted by key. Counter values are increments since
// the previous flush; gauge values are the latest value set.
type Reporter interface {
	// Counter adds a counter delta for the named metric and tags.
	Counter(name string, v int64, tags [][2]string)

	// Gauge sets the gauge value for the named metric and tags.
	Gauge(name string, v float64, tags [][2]string)
}

// RemovableReporter allows a reporter to remove counters and gauges previously
// reported to it.
type RemovableReporter interface {
	// RemoveCounter removes the counter identified by its name and tags.
	RemoveCounter(name string, tags [][2]string)

	// RemoveGauge removes the gauge identified by its name and tags.
	RemoveGauge(name string, tags [][2]string)
}

// HistogramReporter lets a reporter receive histogram observations directly.
type HistogramReporter interface {
	// Histogram returns a function that reports observations for the named
	// histogram and tags.
	Histogram(name string, tags [][2]string) func(v float64)
}

// RemovableHistogramReporter allows a reporter to remove histograms previously
// created through HistogramReporter.
type RemovableHistogramReporter interface {
	// RemoveHistogram removes the histogram identified by its name and tags.
	RemoveHistogram(name string, tags [][2]string)
}

// TimingReporter lets a reporter receive timing observations directly.
type TimingReporter interface {
	// Timing returns a function that reports durations for the named timing and
	// tags.
	Timing(name string, tags [][2]string) func(v time.Duration)
}

// RemovableTimingReporter allows a reporter to remove timings previously
// created through TimingReporter.
type RemovableTimingReporter interface {
	// RemoveTiming removes the timing identified by its name and tags.
	RemoveTiming(name string, tags [][2]string)
}

// Tag is a key/value pair attached to a metric.
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

// Option configures a Statter created by New.
type Option func(*config)

// WithPrefix sets the prefix applied to metric names recorded by the Statter.
func WithPrefix(prefix string) Option {
	return func(c *config) {
		c.prefix = prefix
	}
}

// WithTags sets tags applied to metrics recorded by the Statter. A metric tag
// with the same key overrides the configured value.
func WithTags(tags ...Tag) Option {
	return func(c *config) {
		c.tags = tags
	}
}

// WithSeparator sets the string used to join a prefix and metric name.
func WithSeparator(sep string) Option {
	return func(c *config) {
		c.separator = sep
	}
}

// WithPercentileSamples sets the maximum number of observations retained to
// estimate percentiles for locally aggregated histograms and timings. The
// default is 1000; other aggregate statistics use all observations.
func WithPercentileSamples(n int) Option {
	return func(c *config) {
		c.percSamples = n
	}
}

// WithPercentiles sets the percentiles to report for locally aggregated
// histograms and timings. Each value is a percentile from 0 through 100.
// Percentiles are ignored when the reporter handles observations directly.
func WithPercentiles(p []float64) Option {
	return func(c *config) {
		c.percentiles = p
	}
}

// Statter records metrics and periodically reports them to a Reporter.
// Create a Statter with New; its zero value is not ready to use.
type Statter struct {
	reg    *registry
	prefix string
	tags   []Tag
}

// New returns a Statter that reports metrics to r on each interval tick.
// interval must be positive. Options configure the initial prefix and tags,
// name separator, and percentile aggregation. Call Close on the returned root
// Statter to stop reporting, flush pending metrics, and close r when it
// implements io.Closer.
func New(r Reporter, interval time.Duration, opts ...Option) *Statter {
	cfg := defaultConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	// Sort initial tags once so the root tags match the sorted tag order used
	// for metric keys.
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

// Scope returns a Scope with the given id and tags, derived from s. Metrics
// created through the Scope are tracked by it, and can be removed as a batch
// with [Scope.Delete].
//
// If a Scope with the same id already exists, derived from the same statter and
// with identical tags, it is returned. If it exists with different tags, or was
// derived from a different statter, the existing Scope and every metric created
// through it are deleted and a new Scope is returned. This makes a Scope useful
// for metrics that must be unique for a set of tags, such as a gauge carrying a
// revision that changes over time.
//
// Scope ids are global to the statter tree and are not namespaced by prefix.
//
// A metric created through a Scope shares its registration with an identical
// metric created directly on a statter. Deleting the Scope therefore also
// removes the metric from the direct caller.
func (s *Statter) Scope(id string, tags ...Tag) *Scope {
	return s.reg.Scope(s, id, tags)
}

// HasScope determines if a scope with the given id exists.
func (s *Statter) HasScope(id string) bool {
	return s.reg.HasScope(id)
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

// FullName joins name to the Statter's prefix with the configured separator.
// If the Statter has no prefix, it returns name unchanged.
func (s *Statter) FullName(name string) string {
	if s.prefix != "" {
		return s.prefix + s.reg.cfg.separator + name
	}
	return name
}

// HasCounter reports whether a counter with name and tags has been registered.
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
	c, _ := s.counter(name, tags)
	return c
}

// counter returns the counter for the given name and tags, reporting whether
// this call created it.
func (s *Statter) counter(name string, tags []Tag) (*Counter, bool) {
	k := s.key(name, tags)

	var created bool
	c, ok := s.reg.counters.Load(k.String())
	if !ok {
		n, t := s.mergeDescriptors(name, tags)
		counter := &Counter{
			name: n,
			tags: t,
			key:  k.SafeString(),
			reg:  s.reg,
		}

		var loaded bool
		c, loaded = s.reg.counters.LoadOrStore(k.SafeString(), counter)
		created = !loaded
	}

	k.Release()

	return c, created
}

// HasGauge reports whether a gauge with name and tags has been registered.
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
	g, _ := s.gauge(name, tags)
	return g
}

// gauge returns the gauge for the given name and tags, reporting whether this
// call created it.
func (s *Statter) gauge(name string, tags []Tag) (*Gauge, bool) {
	k := s.key(name, tags)

	var created bool
	g, ok := s.reg.gauges.Load(k.String())
	if !ok {
		n, t := s.mergeDescriptors(name, tags)
		gauge := &Gauge{
			name: n,
			tags: t,
			key:  k.SafeString(),
			reg:  s.reg,
		}

		var loaded bool
		g, loaded = s.reg.gauges.LoadOrStore(k.SafeString(), gauge)
		created = !loaded
	}

	k.Release()

	return g, created
}

// HasHistogram reports whether a histogram with name and tags has been
// registered.
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
	h, _ := s.histogram(name, tags)
	return h
}

// histogram returns the histogram for the given name and tags, reporting
// whether this call created it.
func (s *Statter) histogram(name string, tags []Tag) (*Histogram, bool) {
	k := s.key(name, tags)

	var created bool
	h, ok := s.reg.histograms.Load(k.String())
	if !ok {
		n, t := s.mergeDescriptors(name, tags)
		histogram := newHistogram(s.reg.hr, n, t, s.reg.pool)
		histogram.key = k.SafeString()
		histogram.reg = s.reg

		var loaded bool
		h, loaded = s.reg.histograms.LoadOrStore(k.SafeString(), histogram)
		created = !loaded
	}

	k.Release()

	return h, created
}

// HasTiming reports whether a timing with name and tags has been registered.
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
	t, _ := s.timing(name, tags)
	return t
}

// timing returns the timing for the given name and tags, reporting whether
// this call created it.
func (s *Statter) timing(name string, tags []Tag) (*Timing, bool) {
	k := s.key(name, tags)

	var created bool
	t, ok := s.reg.timings.Load(k.String())
	if !ok {
		n, tags := s.mergeDescriptors(name, tags)
		timing := newTiming(s.reg.tr, n, tags, s.reg.pool)
		timing.key = k.SafeString()
		timing.reg = s.reg

		var loaded bool
		t, loaded = s.reg.timings.LoadOrStore(k.SafeString(), timing)
		created = !loaded
	}

	k.Release()

	return t, created
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

// Close stops the reporting loop, flushes pending metrics, and closes the
// reporter if it implements [io.Closer]. Call Close on the root Statter;
// calling it on a sub-statter returns an error.
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

// Counter accumulates increments between flushes. Each flush reports the
// accumulated delta and resets it to zero.
type Counter struct {
	name string
	tags [][2]string
	key  string
	reg  *registry

	val atomic.Int64
}

// Inc adds v to the counter. The increment may be positive, zero, or negative.
func (c *Counter) Inc(v int64) {
	c.val.Add(v)
}

// Delete stops aggregating the counter and removes it from the reporter when
// the reporter implements RemovableReporter.
func (c *Counter) Delete() {
	c.reg.reportMu.RLock()
	defer c.reg.reportMu.RUnlock()

	if !c.reg.counters.CompareAndDelete(c.key, c) {
		return
	}

	if rr, ok := c.reg.r.(RemovableReporter); ok {
		rr.RemoveCounter(c.name, c.tags)
	}
}

func (c *Counter) value() int64 {
	return c.val.Swap(0)
}

// Gauge holds its last-set value and reports it on each flush.
type Gauge struct {
	name string
	tags [][2]string
	key  string
	reg  *registry

	val atomic.Uint64
}

// Set replaces the gauge value.
func (g *Gauge) Set(v float64) {
	g.val.Store(math.Float64bits(v))
}

// Inc increases the gauge by 1.
func (g *Gauge) Inc() {
	g.Add(1)
}

// Dec decreases the gauge by 1.
func (g *Gauge) Dec() {
	g.Add(-1)
}

// Add adds v to the gauge's current value.
func (g *Gauge) Add(v float64) {
	for {
		oldBits := g.val.Load()
		newBits := math.Float64bits(math.Float64frombits(oldBits) + v)
		if g.val.CompareAndSwap(oldBits, newBits) {
			return
		}
	}
}

// Sub subtracts v from the gauge's current value.
func (g *Gauge) Sub(v float64) {
	g.Add(v * -1)
}

// Delete stops reporting the gauge and removes it from the reporter when the
// reporter implements RemovableReporter.
func (g *Gauge) Delete() {
	g.reg.reportMu.RLock()
	defer g.reg.reportMu.RUnlock()

	if !g.reg.gauges.CompareAndDelete(g.key, g) {
		return
	}

	if rr, ok := g.reg.r.(RemovableReporter); ok {
		rr.RemoveGauge(g.name, g.tags)
	}
}

func (g *Gauge) value() float64 {
	v := g.val.Load()
	return math.Float64frombits(v)
}

// Histogram records observations for a metric.
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

// Observe adds v to the histogram.
func (h *Histogram) Observe(v float64) {
	if h.hrFn != nil {
		h.hrFn(v)
		return
	}

	h.mu.Lock()
	h.s.Add(v)
	h.mu.Unlock()
}

// Delete stops recording the histogram and removes it from the reporter when
// the reporter implements RemovableHistogramReporter or RemovableReporter.
func (h *Histogram) Delete() {
	h.reg.reportMu.RLock()
	defer h.reg.reportMu.RUnlock()

	if !h.reg.histograms.CompareAndDelete(h.key, h) {
		return
	}

	if rtr, ok := h.reg.r.(RemovableHistogramReporter); ok {
		rtr.RemoveHistogram(h.name, h.tags)
	} else if rr, ok := h.reg.r.(RemovableReporter); ok {
		rr.RemoveCounter(countKey(h.name), h.tags)
		for _, k := range h.reg.sampleGaugeKeys(h.name, "") {
			rr.RemoveGauge(k, h.tags)
		}
	}
}

func (h *Histogram) value() *stats.Sample {
	h.mu.Lock()
	s := h.s
	h.s = h.pool.Get()
	h.mu.Unlock()

	return s
}

// Timing records duration observations for a metric.
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

// Observe adds d to the timing. When the reporter does not handle timings,
// durations are aggregated and reported in milliseconds.
func (t *Timing) Observe(d time.Duration) {
	if t.trFn != nil {
		t.trFn(d)
		return
	}

	t.mu.Lock()
	t.s.Add(d.Seconds() * 1000)
	t.mu.Unlock()
}

// Delete stops recording the timing and removes it from the reporter when the
// reporter implements RemovableTimingReporter or RemovableReporter.
func (t *Timing) Delete() {
	t.reg.reportMu.RLock()
	defer t.reg.reportMu.RUnlock()

	if !t.reg.timings.CompareAndDelete(t.key, t) {
		return
	}

	if rtr, ok := t.reg.r.(RemovableTimingReporter); ok {
		rtr.RemoveTiming(t.name, t.tags)
	} else if rr, ok := t.reg.r.(RemovableReporter); ok {
		rr.RemoveCounter(countKey(t.name), t.tags)
		for _, k := range t.reg.sampleGaugeKeys(t.name, "_ms") {
			rr.RemoveGauge(k, t.tags)
		}
	}
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
