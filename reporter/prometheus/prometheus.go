// Package prometheus implements an prometheus stats client.
package prometheus

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Option represents statsd option function.
type Option func(p *Prometheus)

// WithBuckets sets the buckets to used with histograms.
func WithBuckets(buckets []float64) Option {
	return func(p *Prometheus) {
		p.buckets = buckets
	}
}

// Prometheus is a prometheus stats collector.
type Prometheus struct {
	namespace string

	fqn *fqn

	buckets    []float64
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
		namespace: fqn.Format(namespace),
		fqn:       fqn,
		buckets:   prometheus.DefBuckets,
		reg:       prometheus.NewRegistry(),
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
		histo := prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: p.namespace,
				Name:      p.fqn.Format(name),
				Buckets:   p.buckets,
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
		timing := prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: p.namespace,
				Name:      p.fqn.Format(name),
				Buckets:   p.buckets,
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

// Close closes the client and flushes buffered stats, if applicable.
func (p *Prometheus) Close() error {
	return nil
}

// createKey creates a unique metric key.
func createKey(name string, lblNames []string) string {
	return name + strings.Join(lblNames, ":")
}

// formatTags create a prometheus Label map from tags.
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
