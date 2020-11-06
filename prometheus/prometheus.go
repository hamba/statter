// Package prometheus implements an prometheus stats client.
package prometheus

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/hamba/statter/internal/bytes"
	"github.com/hamba/statter/internal/tags"
)

// Logger represents an abstract logging object.
type Logger interface {
	Error(msg string, ctx ...interface{})
}

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

	set      *metrics.Set
	counters map[string]*metrics.Counter
	gauges   map[string]*gauge
	timings  map[string]*metrics.Summary
}

// New returns a new prometheus stats instance.
func New(prefix string) *Prometheus {
	fqn := NewFQN()

	return &Prometheus{
		prefix:   fqn.Format(prefix),
		fqn:      fqn,
		set:      metrics.NewSet(),
		counters: map[string]*metrics.Counter{},
		gauges:   map[string]*gauge{},
		timings:  map[string]*metrics.Summary{},
	}
}

// Handler gets the prometheus HTTP handler.
func (s *Prometheus) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.set.WritePrometheus(w)
	})
}

// Inc increments a count by the value.
func (s *Prometheus) Inc(name string, value int64, rate float32, tags ...string) {
	lbls := formatTags(tags, s.fqn)
	key := createKey(s.prefix, name, lbls, s.fqn)
	m, ok := s.counters[key]
	if !ok {
		m = s.set.NewCounter(key)
		s.counters[key] = m
	}

	m.Add(int(value))
}

// Dec decrements a count by the value.
func (s *Prometheus) Dec(name string, value int64, rate float32, tags ...string) {
	lbls := formatTags(tags, s.fqn)
	key := createKey(s.prefix, name, lbls, s.fqn)
	m, ok := s.counters[key]
	if !ok {
		m = s.set.NewCounter(key)
		s.counters[key] = m
	}

	m.Set(m.Get() - uint64(value))
}

type gauge struct {
	mu sync.Mutex
	f  float64
}

func (g *gauge) Get() float64 {
	g.mu.Lock()
	f := g.f
	g.mu.Unlock()
	return f
}

func (g *gauge) Set(f float64) {
	g.mu.Lock()
	g.f = f
	g.mu.Unlock()
}

// Gauge measures the value of a metric.
func (s *Prometheus) Gauge(name string, value float64, rate float32, tags ...string) {
	lbls := formatTags(tags, s.fqn)
	key := createKey(s.prefix, name, lbls, s.fqn)
	g, ok := s.gauges[key]
	if !ok {
		g = &gauge{}
		_ = s.set.NewGauge(key, g.Get)
		s.gauges[key] = g
	}

	g.Set(value)
}

// Timing sends the value of a Duration.
func (s *Prometheus) Timing(name string, value time.Duration, rate float32, tags ...string) {
	lbls := formatTags(tags, s.fqn)
	key := createKey(s.prefix, name, lbls, s.fqn)
	m, ok := s.timings[key]
	if !ok {
		m = s.set.NewSummary(key)
		s.timings[key] = m
	}

	m.Update(float64(value) / float64(time.Millisecond))
}

// Close closes the client and flushes buffered stats, if applicable.
func (s *Prometheus) Close() error {
	return nil
}

// createKey creates a unique metric key.
func createKey(prefix, name, lbls string, fqn *FQN) string {
	if len(lbls) == 0 {
		return prefix + "_" + fqn.Format(name)
	}
	return prefix + "_" + fqn.Format(name) + "{" + lbls + "}"
}

var pool = bytes.NewPool(512)

// formatTags create a prometheus Label map from tags.
func formatTags(t []string, fqn *FQN) string {
	if len(t) == 0 {
		return ""
	}

	t = tags.Deduplicate(tags.Normalize(t))

	buf := pool.Get()
	for i := 0; i < len(t); i += 2 {
		if i > 0 {
			_ = buf.WriteByte(',')
		}
		buf.WriteString(fqn.Format(t[i]))
		_ = buf.WriteByte('=')
		_ = buf.WriteByte('"')
		buf.WriteString(t[i+1])
		_ = buf.WriteByte('"')
	}

	s := string(buf.Bytes())
	pool.Put(buf)
	return s
}
