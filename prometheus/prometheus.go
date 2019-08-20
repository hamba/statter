package prometheus

import (
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/hamba/statter/internal/tags"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// FQN is a name formatter.
type FQN struct {
	r *strings.Replacer
}

// NewFQN returns an FQN formatter.
func NewFQN() *FQN {
	return &FQN{
		r: strings.NewReplacer(".", "_", "-", "_"),
	}
}

// Format formats a name to be a fully qualified name.
func (f *FQN) Format(name string) string {
	return f.r.Replace(name)
}

// Prometheus is a prometheus stats collector.
type Prometheus struct {
	prefix string

	fqn *FQN

	reg      *prometheus.Registry
	counters map[string]*prometheus.CounterVec
	gauges   map[string]*prometheus.GaugeVec
	timings  map[string]*prometheus.SummaryVec
}

// New returns a new prometheus stats instance.
func New(prefix string) *Prometheus {
	fqn := NewFQN()

	return &Prometheus{
		prefix:   fqn.Format(prefix),
		fqn:      fqn,
		reg:      prometheus.NewRegistry(),
		counters: map[string]*prometheus.CounterVec{},
		gauges:   map[string]*prometheus.GaugeVec{},
		timings:  map[string]*prometheus.SummaryVec{},
	}
}

// Handler gets the prometheus HTTP handler.
func (s *Prometheus) Handler() http.Handler {
	return promhttp.HandlerFor(s.reg, promhttp.HandlerOpts{})
}

// Inc increments a count by the value.
func (s *Prometheus) Inc(name string, value int64, rate float32, tags ...string) error {
	lblNames, lbls := formatTags(tags)

	key := createKey(name, lblNames)
	m, ok := s.counters[key]
	if !ok {
		m = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: s.prefix,
				Name:      s.fqn.Format(name),
				Help:      name,
			},
			lblNames,
		)

		if err := s.reg.Register(m); err != nil {
			return err
		}
		s.counters[key] = m
	}

	m.With(lbls).Add(float64(value))

	return nil
}

// Dec decrements a count by the value.
func (s *Prometheus) Dec(name string, value int64, rate float32, tags ...string) error {
	return errors.New("prometheus: decrement not supported")
}

// Gauge measures the value of a metric.
func (s *Prometheus) Gauge(name string, value float64, rate float32, tags ...string) error {
	lblNames, lbls := formatTags(tags)

	key := createKey(name, lblNames)
	m, ok := s.gauges[key]
	if !ok {
		m = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: s.prefix,
				Name:      s.fqn.Format(name),
				Help:      name,
			},
			lblNames,
		)

		if err := s.reg.Register(m); err != nil {
			return err
		}
		s.gauges[key] = m
	}

	m.With(lbls).Set(value)

	return nil
}

// Timing sends the value of a Duration.
func (s *Prometheus) Timing(name string, value time.Duration, rate float32, tags ...string) error {
	lblNames, lbls := formatTags(tags)

	key := createKey(name, lblNames)
	m, ok := s.timings[key]
	if !ok {
		m = prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace:  s.prefix,
				Name:       s.fqn.Format(name),
				Help:       name,
				Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
			},
			lblNames,
		)

		if err := s.reg.Register(m); err != nil {
			return err
		}
		s.timings[key] = m
	}

	m.With(lbls).Observe(float64(value) / float64(time.Millisecond))

	return nil
}

// Close closes the client and flushes buffered stats, if applicable
func (s *Prometheus) Close() error {
	return nil
}

// createKey creates a unique metric key.
func createKey(name string, lblNames []string) string {
	return name + strings.Join(lblNames, ":")
}

// formatTags create a prometheus Label map from tags.
func formatTags(t []string) ([]string, prometheus.Labels) {
	t = tags.Deduplicate(tags.Normalize(t))

	names := make([]string, 0, len(t)/2)
	lbls := make(prometheus.Labels, len(t)/2)
	for i := 0; i < len(t); i += 2 {
		key := t[i]
		names = append(names, key)
		lbls[key] = t[i+1]
	}

	sort.Strings(names)

	return names, lbls
}
