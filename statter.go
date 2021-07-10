package statter

import (
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hamba/statter/internal/stats"
)

// NullReporter is a reporter that discards all stats.
var NullReporter = nullReporter{}

// Reporter represents a stats reporter.
type Reporter interface {
	Counter(name string, v int64, tags [][2]string)
	Gauge(name string, v float64, tags [][2]string)
}

// HistogramReporter represents a stats reporter that handles histograms.
type HistogramReporter interface {
	Histogram(name string, tags [][2]string) func(v float64)
}

// TimingReporter represents a stats reporter that handles timings.
type TimingReporter interface {
	Timing(name string, tags [][2]string) func(v time.Duration)
}

// Tag is a stat tag.
type Tag [2]string

type config struct {
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
	hr   HistogramReporter
	tr   TimingReporter
	pool *stats.Pool

	prefix string
	tags   []Tag

	counters   counterMap
	gauges     gaugeMap
	histograms histogramMap
	timings    timingMap
}

// New returns a statter.
func New(r Reporter, interval time.Duration, opts ...Option) *Statter {
	cfg := defaultConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	s := &Statter{
		cfg:  cfg,
		r:    r,
		pool: stats.NewPool(cfg.percSamples),
	}
	s.reg = newRegistry(s, interval)

	if hr, ok := r.(HistogramReporter); ok {
		s.hr = hr
	}
	if tr, ok := r.(TimingReporter); ok {
		s.tr = tr
	}

	return s
}

// With returns a statter with the given prefix and tags.
func (s *Statter) With(prefix string, tags ...Tag) *Statter {
	return s.reg.SubStatter(s, prefix, tags)
}

// Counter returns a counter for the given name and tags.
func (s *Statter) Counter(name string, tags ...Tag) *Counter {
	k := newKey(name, tags)

	c, ok := s.counters.Load(k.String())
	if !ok {
		n, t := s.mergeDescriptors(name, tags)
		counter := &Counter{
			name: n,
			tags: t,
		}
		c, _ = s.counters.LoadOrStore(k.String(), counter)
	}

	putKey(k)

	return c
}

// Gauge returns a gauge for the given name and tags.
func (s *Statter) Gauge(name string, tags ...Tag) *Gauge {
	k := newKey(name, tags)

	g, ok := s.gauges.Load(k.String())
	if !ok {
		n, t := s.mergeDescriptors(name, tags)
		gauge := &Gauge{
			name: n,
			tags: t,
		}
		g, _ = s.gauges.LoadOrStore(k.String(), gauge)
	}

	putKey(k)

	return g
}

// Histogram returns a histogram for the given name and tags.
func (s *Statter) Histogram(name string, tags ...Tag) *Histogram {
	k := newKey(name, tags)

	h, ok := s.histograms.Load(k.String())
	if !ok {
		n, t := s.mergeDescriptors(name, tags)
		histogram := newHistogram(s.hr, n, t, s.pool)
		h, _ = s.histograms.LoadOrStore(k.String(), histogram)
	}

	putKey(k)

	return h
}

// Timing returns a timing for the given name and tags.
func (s *Statter) Timing(name string, tags ...Tag) *Timing {
	k := newKey(name, tags)

	t, ok := s.timings.Load(k.String())
	if !ok {
		n, newTags := s.mergeDescriptors(name, tags)
		timing := newTiming(s.tr, n, newTags, s.pool)
		t, _ = s.timings.LoadOrStore(k.String(), timing)
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

	if s.hr == nil {
		s.histograms.Range(func(_ string, h *Histogram) bool {
			histo := h.value()
			defer s.pool.Put(histo)

			s.reportSample(h.name, "", h.tags, histo)
			return true
		})
	}

	if s.tr == nil {
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
	for i := 0; i < len(vs); i++ {
		name := prefix + strconv.FormatFloat(ps[i], 'g', -1, 64) + "p" + suffix
		s.r.Gauge(name, vs[i], tags)
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

// Close closes the statter.
func (s *Statter) Close() error {
	return s.reg.Close(s)
}

// Counter implements a counter.
type Counter struct {
	name string
	tags [][2]string

	val int64
}

// Inc increments the counter.
func (c *Counter) Inc(v int64) {
	atomic.AddInt64(&c.val, v)
}

func (c *Counter) value() int64 {
	return atomic.SwapInt64(&c.val, 0)
}

// Gauge implements a gauge.
type Gauge struct {
	name string
	tags [][2]string

	val uint64
}

// Set sets the gauge value.
func (g *Gauge) Set(v float64) {
	atomic.StoreUint64(&g.val, math.Float64bits(v))
}

func (g *Gauge) value() float64 {
	v := atomic.SwapUint64(&g.val, 0)
	return math.Float64frombits(v)
}

// Histogram implements a histogram.
type Histogram struct {
	hrFn func(v float64)
	name string
	tags [][2]string
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

func (t *Timing) value() *stats.Sample {
	t.mu.Lock()
	s := t.s
	t.s = t.pool.Get()
	t.mu.Unlock()

	return s
}

type nullReporter struct{}

func (r nullReporter) Counter(_ string, _ int64, _ [][2]string) {}
func (r nullReporter) Gauge(_ string, _ float64, _ [][2]string) {}

func (r nullReporter) Histogram(_ string, _ [][2]string) func(float64) {
	return func(_ float64) {}
}

func (r nullReporter) Timing(_ string, _ [][2]string) func(time.Duration) {
	return func(_ time.Duration) {}
}
