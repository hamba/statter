// Package prometheus implements an prometheus stats client.
package prometheus

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/hamba/statter/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Option represents statsd option function.
type Option func(p *Prometheus)

// WithBuckets sets the buckets to used with histograms.
func WithBuckets(buckets []float64) Option {
	return func(p *Prometheus) {
		p.defBuckets = buckets
	}
}

// Prometheus is a prometheus stats collector.
type Prometheus struct {
	namespace string

	fqn *fqn

	defBuckets []float64
	buckets    bucketMap

	reg        *prometheus.Registry
	counters   counterMap
	gauges     gaugeMap
	histograms histogramMap
	timings    histogramMap
}

// New returns a new prometheus reporter.
func New(namespace string, opts ...Option) *Prometheus {
	fqn := newFQN()

	p := &Prometheus{
		namespace:  fqn.Format(namespace),
		fqn:        fqn,
		defBuckets: prometheus.DefBuckets,
		reg:        prometheus.NewRegistry(),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Handler gets the prometheus HTTP handler.
func (p *Prometheus) Handler() http.Handler {
	return promhttp.HandlerFor(p.reg, promhttp.HandlerOpts{})
}

// Counter reports a counter value.
func (p *Prometheus) Counter(name string, v int64, tags [][2]string) {
	lblNames, lbls := formatTags(tags, p.fqn)
	key := createKey(name, lblNames)

	m, ok := p.counters.Load(key)
	if !ok {
		counter := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: p.namespace,
				Name:      p.fqn.Format(name),
				Help:      name,
			},
			lblNames,
		)

		m, ok = p.counters.LoadOrStore(key, counter)
		if !ok {
			_ = p.reg.Register(m)
		}
	}

	m.With(lbls).Add(float64(v))
}

// Gauge reports a gauge value.
func (p *Prometheus) Gauge(name string, v float64, tags [][2]string) {
	lblNames, lbls := formatTags(tags, p.fqn)
	key := createKey(name, lblNames)

	m, ok := p.gauges.Load(key)
	if !ok {
		gauge := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: p.namespace,
				Name:      p.fqn.Format(name),
				Help:      name,
			},
			lblNames,
		)

		m, ok = p.gauges.LoadOrStore(key, gauge)
		if !ok {
			_ = p.reg.Register(m)
		}
	}

	m.With(lbls).Set(v)
}

// Histogram reports a histogram value.
func (p *Prometheus) Histogram(name string, tags [][2]string) func(v float64) {
	lblNames, lbls := formatTags(tags, p.fqn)
	key := createKey(name, lblNames)

	m, ok := p.histograms.Load(key)
	if !ok {
		buckets := p.getBuckets(name)
		histo := prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: p.namespace,
				Name:      p.fqn.Format(name),
				Buckets:   buckets,
				Help:      name,
			},
			lblNames,
		)

		m, ok = p.histograms.LoadOrStore(key, histo)
		if !ok {
			_ = p.reg.Register(m)
		}
	}

	o := m.With(lbls)
	return func(v float64) {
		o.Observe(v)
	}
}

// Timing reports a timing value as a histogram in seconds.
func (p *Prometheus) Timing(name string, tags [][2]string) func(v time.Duration) {
	lblNames, lbls := formatTags(tags, p.fqn)
	key := createKey(name, lblNames)

	m, ok := p.timings.Load(key)
	if !ok {
		buckets := p.getBuckets(name)
		timing := prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: p.namespace,
				Name:      p.fqn.Format(name),
				Buckets:   buckets,
				Help:      name,
			},
			lblNames,
		)

		m, ok = p.timings.LoadOrStore(key, timing)
		if !ok {
			_ = p.reg.Register(m)
		}
	}

	o := m.With(lbls)
	return func(v time.Duration) {
		o.Observe(v.Seconds())
	}
}

func (p *Prometheus) getBuckets(name string) []float64 {
	b, ok := p.buckets.Load(name)
	if !ok {
		return p.defBuckets
	}
	return b
}

// Close closes the client and flushes buffered stats, if applicable.
func (p *Prometheus) Close() error {
	return nil
}

// SetMetricBuckets sets the buckets for a metric by name.
//
// This must be called before the metric is used, otherwise will be
// ignored or can have unexpected results.
//
// Deprecated: Use RegisterHistogram instead, this function will be removed in a future release.
func SetMetricBuckets(stats *statter.Statter, name string, buckets []float64) {
	prom, ok := stats.Reporter().(*Prometheus)
	if !ok {
		return
	}

	prom.buckets.Store(stats.FullName(name), buckets)
}

// RegisterCounter registers a counter with the given label names with the prometheus registrar,
// returning false if it has already been registered.
func RegisterCounter(stats *statter.Statter, name string, lblNames []string, help string) bool {
	prom, ok := stats.Reporter().(*Prometheus)
	if !ok {
		return true
	}

	name = stats.FullName(name)
	sort.Strings(lblNames)

	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: prom.namespace,
			Name:      prom.fqn.Format(name),
			Help:      help,
		},
		lblNames,
	)

	key := createKey(name, lblNames)
	if vec, ok := prom.counters.LoadOrStore(key, counter); !ok {
		_ = prom.reg.Register(vec)
		return true
	}
	return false
}

// RegisterGauge registers a gauge with the given label names with the prometheus registrar,
// returning false if it has already been registered.
func RegisterGauge(stats *statter.Statter, name string, lblNames []string, help string) bool {
	prom, ok := stats.Reporter().(*Prometheus)
	if !ok {
		return true
	}

	name = stats.FullName(name)
	sort.Strings(lblNames)

	gauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: prom.namespace,
			Name:      prom.fqn.Format(name),
			Help:      help,
		},
		lblNames,
	)

	key := createKey(name, lblNames)
	if vec, ok := prom.gauges.LoadOrStore(key, gauge); !ok {
		_ = prom.reg.Register(vec)
		return true
	}
	return false
}

// RegisterHistogram registers a histogram with the given label names and buckets with the prometheus registrar,
// returning false if it has already been registered.
func RegisterHistogram(stats *statter.Statter, name string, lblNames []string, buckets []float64, help string) bool {
	prom, ok := stats.Reporter().(*Prometheus)
	if !ok {
		return true
	}

	name = stats.FullName(name)
	sort.Strings(lblNames)
	if len(buckets) == 0 {
		buckets = prom.defBuckets
	}

	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: prom.namespace,
			Name:      prom.fqn.Format(name),
			Buckets:   buckets,
			Help:      help,
		},
		lblNames,
	)

	key := createKey(name, lblNames)
	if vec, ok := prom.histograms.LoadOrStore(key, histogram); !ok {
		_ = prom.reg.Register(vec)
		return true
	}
	return false
}

// createKey creates a unique metric key.
func createKey(name string, lblNames []string) string {
	return name + strings.Join(lblNames, ":")
}

// formatTags creates a prometheus Label map from tags.
func formatTags(tags [][2]string, fqn *fqn) ([]string, prometheus.Labels) {
	names := make([]string, 0, len(tags))
	lbls := make(prometheus.Labels, len(tags))
	for _, tag := range tags {
		key := fqn.Format(tag[0])
		names = append(names, key)
		lbls[key] = tag[1]
	}

	sort.Strings(names)

	return names, lbls
}

type fqn struct {
	r *strings.Replacer
}

func newFQN() *fqn {
	return &fqn{
		r: strings.NewReplacer(".", "_", "-", "_"),
	}
}

func (f *fqn) Format(name string) string {
	return f.r.Replace(name)
}
