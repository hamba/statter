package statter

import (
	"io"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go4org/hashtriemap"
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
type Tag [2]string

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
	cfg config
	reg *registry

	r    Reporter
	hr   *value[HistogramReporter]
	tr   *value[TimingReporter]
	pool *stats.Pool

	prefix string
	tags   []Tag

	counters   hashtriemap.HashTrieMap[string, *Counter]
	gauges     hashtriemap.HashTrieMap[string, *Gauge]
	histograms hashtriemap.HashTrieMap[string, *Histogram]
	timings    hashtriemap.HashTrieMap[string, *Timing]
}

// New returns a statter.
func New(r Reporter, interval time.Duration, opts ...Option) *Statter {
	cfg := defaultConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	s := &Statter{
		cfg:    cfg,
		r:      r,
		hr:     &value[HistogramReporter]{},
		tr:     &value[TimingReporter]{},
		pool:   stats.NewPool(cfg.percSamples),
		prefix: cfg.prefix,
		tags:   cfg.tags,
	}
	s.reg = newRegistry(s, interval)

	if hr, ok := r.(HistogramReporter); ok {
		s.hr.Store(hr)
	}
	if tr, ok := r.(TimingReporter); ok {
		s.tr.Store(tr)
	}

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
	return s.r
}

// FullName returns the full name with prefix for the given name.
func (s *Statter) FullName(name string) string {
	if s.prefix != "" {
		return s.prefix + s.cfg.separator + name
	}
	return name
}

// HasCounter determines if the counter exists.
func (s *Statter) HasCounter(name string, tags ...Tag) bool {
	k := newKey(name, tags)

	_, ok := s.counters.Load(k.String())

	putKey(k)

	return ok
}

// Counter returns a counter for the given name and tags.
func (s *Statter) Counter(name string, tags ...Tag) *Counter {
	k := newKey(name, tags)

	c, ok := s.counters.Load(k.String())
	if !ok {
		n, t := s.mergeDescriptors(name, tags)
		counter := &Counter{
			name:     n,
			tags:     t,
			deleteFn: s.deleteCounterFunc(k.SafeString(), n, t),
		}
		c, _ = s.counters.LoadOrStore(k.SafeString(), counter)
	}

	putKey(k)

	return c
}

// HasGauge determines if the gauge exists.
func (s *Statter) HasGauge(name string, tags ...Tag) bool {
	k := newKey(name, tags)

	_, ok := s.gauges.Load(k.String())

	putKey(k)

	return ok
}

// Gauge returns a gauge for the given name and tags.
func (s *Statter) Gauge(name string, tags ...Tag) *Gauge {
	k := newKey(name, tags)

	g, ok := s.gauges.Load(k.String())
	if !ok {
		n, t := s.mergeDescriptors(name, tags)
		gauge := &Gauge{
			name:     n,
			tags:     t,
			deleteFn: s.deleteGaugeFunc(k.SafeString(), n, t),
		}
		g, _ = s.gauges.LoadOrStore(k.SafeString(), gauge)
	}

	putKey(k)

	return g
}

// HasHistogram determines if the histogram exists.
func (s *Statter) HasHistogram(name string, tags ...Tag) bool {
	k := newKey(name, tags)

	_, ok := s.histograms.Load(k.String())

	putKey(k)

	return ok
}

// Histogram returns a histogram for the given name and tags.
func (s *Statter) Histogram(name string, tags ...Tag) *Histogram {
	k := newKey(name, tags)

	h, ok := s.histograms.Load(k.String())
	if !ok {
		n, t := s.mergeDescriptors(name, tags)
		histogram := newHistogram(s.hr.Load(), n, t, s.pool)
		histogram.deleteFn = s.deleteHistogramFunc(k.SafeString(), n, t)
		h, _ = s.histograms.LoadOrStore(k.SafeString(), histogram)
	}

	putKey(k)

	return h
}

// HasTiming determines if the timing exists.
func (s *Statter) HasTiming(name string, tags ...Tag) bool {
	k := newKey(name, tags)

	_, ok := s.timings.Load(k.String())

	putKey(k)

	return ok
}

// Timing returns a timing for the given name and tags.
func (s *Statter) Timing(name string, tags ...Tag) *Timing {
	k := newKey(name, tags)

	t, ok := s.timings.Load(k.String())
	if !ok {
		n, newTags := s.mergeDescriptors(name, tags)
		timing := newTiming(s.tr.Load(), n, newTags, s.pool)
		timing.deleteFn = s.deleteTimingFunc(k.SafeString(), n, newTags)
		t, _ = s.timings.LoadOrStore(k.SafeString(), timing)
	}

	putKey(k)

	return t
}

func (s *Statter) report() {
	s.counters.Range(func(_ string, c *Counter) bool {
		val := c.value()
		if val == 0 {
			return true
		}
		s.r.Counter(c.name, val, c.tags)
		return true
	})

	s.gauges.Range(func(_ string, g *Gauge) bool {
		s.r.Gauge(g.name, g.value(), g.tags)
		return true
	})

	if s.hr.Load() == nil {
		s.histograms.Range(func(_ string, h *Histogram) bool {
			histo := h.value()
			defer s.pool.Put(histo)

			s.reportSample(h.name, "", h.tags, histo)
			return true
		})
	}

	if s.tr.Load() == nil {
		s.timings.Range(func(_ string, t *Timing) bool {
			timing := t.value()
			defer s.pool.Put(timing)

			s.reportSample(t.name, "_ms", t.tags, timing)
			return true
		})
	}
}

func (s *Statter) reportSample(name, suffix string, tags [][2]string, sample *stats.Sample) {
	if sample.Count() == 0 {
		return
	}

	prefix := name + "_"
	s.r.Counter(prefix+"count", sample.Count(), tags)
	s.r.Gauge(prefix+"sum"+suffix, sample.Sum(), tags)
	s.r.Gauge(prefix+"mean"+suffix, sample.Mean(), tags)
	s.r.Gauge(prefix+"stddev"+suffix, sample.StdDev(), tags)
	s.r.Gauge(prefix+"min"+suffix, sample.Min(), tags)
	s.r.Gauge(prefix+"max"+suffix, sample.Max(), tags)
	ps := s.cfg.percentiles
	vs := sample.Percentiles(ps)
	for i := range vs {
		name := prefix + strconv.FormatFloat(ps[i], 'g', -1, 64) + "p" + suffix
		s.r.Gauge(name, vs[i], tags)
	}
}

func (s *Statter) sampleKeys(name, suffix string) []string {
	prefix := name + "_"
	keys := make([]string, 0, 6+len(s.cfg.percentiles))
	keys = append(keys, prefix+"count")
	keys = append(keys, prefix+"sum"+suffix)
	keys = append(keys, prefix+"mean"+suffix)
	keys = append(keys, prefix+"stddev"+suffix)
	keys = append(keys, prefix+"min"+suffix)
	keys = append(keys, prefix+"max"+suffix)

	for _, p := range s.cfg.percentiles {
		keys = append(keys, prefix+strconv.FormatFloat(p, 'g', -1, 64)+"p"+suffix)
	}

	return keys
}

func (s *Statter) deleteCounterFunc(key, name string, tags [][2]string) func() {
	return func() {
		if rr, ok := s.r.(RemovableReporter); ok {
			rr.RemoveCounter(name, tags)
		}
		_, _ = s.counters.LoadAndDelete(key)
	}
}

func (s *Statter) deleteGaugeFunc(key, name string, tags [][2]string) func() {
	return func() {
		if rr, ok := s.r.(RemovableReporter); ok {
			rr.RemoveGauge(name, tags)
		}
		_, _ = s.gauges.LoadAndDelete(key)
	}
}

func (s *Statter) deleteHistogramFunc(key, name string, tags [][2]string) func() {
	return func() {
		if rtr, ok := s.r.(RemovableHistogramReporter); ok {
			rtr.RemoveHistogram(name, tags)
		} else if rr, ok := s.r.(RemovableReporter); ok {
			keys := s.sampleKeys(name, "")
			for _, k := range keys {
				rr.RemoveGauge(k, tags)
			}
		}
		_, _ = s.histograms.LoadAndDelete(key)
	}
}

func (s *Statter) deleteTimingFunc(key, name string, tags [][2]string) func() {
	return func() {
		if rtr, ok := s.r.(RemovableTimingReporter); ok {
			rtr.RemoveTiming(name, tags)
		} else if rr, ok := s.r.(RemovableReporter); ok {
			keys := s.sampleKeys(name, "")
			for _, k := range keys {
				rr.RemoveGauge(k, tags)
			}
		}
		_, _ = s.timings.LoadAndDelete(key)
	}
}

func (s *Statter) mergeDescriptors(name string, tags []Tag) (string, [][2]string) {
	if s.prefix != "" {
		name = s.prefix + s.cfg.separator + name
	}

	newTags := make([][2]string, 0, len(s.tags)+len(tags))
	for _, tag := range s.tags {
		newTags = append(newTags, tag)
	}
	for _, tag := range tags {
		newTags = append(newTags, tag)
	}

	return name, newTags
}

// Close closes the statter and reporter.
func (s *Statter) Close() error {
	if err := s.reg.Close(s); err != nil {
		return err
	}

	if c, ok := s.r.(io.Closer); ok {
		if err := c.Close(); err != nil {
			return err
		}
	}

	return nil
}

// Counter implements a counter.
type Counter struct {
	name     string
	tags     [][2]string
	deleteFn func()

	val int64
}

// Inc increments the counter.
func (c *Counter) Inc(v int64) {
	atomic.AddInt64(&c.val, v)
}

// Delete removes the counter.
func (c *Counter) Delete() {
	c.deleteFn()
}

func (c *Counter) value() int64 {
	return atomic.SwapInt64(&c.val, 0)
}

// Gauge implements a gauge.
type Gauge struct {
	name     string
	tags     [][2]string
	deleteFn func()

	val uint64
}

// Set sets the gauge value.
func (g *Gauge) Set(v float64) {
	atomic.StoreUint64(&g.val, math.Float64bits(v))
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
		oldBits := atomic.LoadUint64(&g.val)
		newBits := math.Float64bits(math.Float64frombits(oldBits) + v)
		if atomic.CompareAndSwapUint64(&g.val, oldBits, newBits) {
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
	g.deleteFn()
}

func (g *Gauge) value() float64 {
	v := atomic.LoadUint64(&g.val)
	return math.Float64frombits(v)
}

// Histogram implements a histogram.
type Histogram struct {
	hrFn     func(v float64)
	name     string
	tags     [][2]string
	deleteFn func()
	pool     *stats.Pool

	mu sync.Mutex
	s  *stats.Sample
}

func newHistogram(hr HistogramReporter, name string, tags [][2]string, pool *stats.Pool) *Histogram {
	if hr != nil {
		fn := hr.Histogram(name, tags)
		if fn != nil {
			return &Histogram{
				hrFn: fn,
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
	h.deleteFn()
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
	trFn     func(v time.Duration)
	name     string
	tags     [][2]string
	deleteFn func()
	pool     *stats.Pool

	mu sync.Mutex
	s  *stats.Sample
}

func newTiming(tr TimingReporter, name string, tags [][2]string, pool *stats.Pool) *Timing {
	if tr != nil {
		fn := tr.Timing(name, tags)
		if fn != nil {
			return &Timing{
				trFn: fn,
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
	t.deleteFn()
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
