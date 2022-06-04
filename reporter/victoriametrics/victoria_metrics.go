// Package victoriametrics implements an victoria metrics stats reporter.
package victoriametrics

import (
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/hamba/statter/v2/internal/bytes"
)

// VictoriaMetrics is a victoria metrics stats reporter.
type VictoriaMetrics struct {
	fqn *fqn

	mu     sync.RWMutex
	gauges map[string]*gauge

	set *metrics.Set
}

// New returns a new victoria metrics reporter.
func New() *VictoriaMetrics {
	fqn := newFQN()

	return &VictoriaMetrics{
		fqn:    fqn,
		set:    metrics.NewSet(),
		gauges: map[string]*gauge{},
	}
}

// Handler gets the victoria metrics HTTP handler.
func (m *VictoriaMetrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.set.WritePrometheus(w)
	})
}

// Counter reports a counter value.
func (m *VictoriaMetrics) Counter(name string, v int64, tags [][2]string) {
	lbls := formatTags(tags, m.fqn)
	key := createKey(name, lbls, m.fqn)

	c := m.set.GetOrCreateCounter(key)

	c.Add(int(v))
}

type gauge struct {
	val uint64
}

func (g *gauge) Get() float64 {
	v := atomic.LoadUint64(&g.val)
	return math.Float64frombits(v)
}

func (g *gauge) Set(v float64) {
	atomic.StoreUint64(&g.val, math.Float64bits(v))
}

// Gauge reports a gauge value.
func (m *VictoriaMetrics) Gauge(name string, v float64, tags [][2]string) {
	lbls := formatTags(tags, m.fqn)
	key := createKey(name, lbls, m.fqn)

	if m.setExistingGauge(key, v) {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double check that it was not added while we queued for the lock.
	g, ok := m.gauges[key]
	if ok {
		g.Set(v)
		return
	}

	g = &gauge{}
	m.gauges[key] = g

	m.set.NewGauge(key, g.Get)

	g.Set(v)
}

func (m *VictoriaMetrics) setExistingGauge(key string, v float64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	g, ok := m.gauges[key]
	if ok {
		g.Set(v)
		return true
	}
	return false
}

// Histogram reports a histogram value.
func (m *VictoriaMetrics) Histogram(name string, tags [][2]string) func(v float64) {
	lbls := formatTags(tags, m.fqn)
	key := createKey(name, lbls, m.fqn)

	h := m.set.GetOrCreateHistogram(key)

	return func(v float64) {
		h.Update(v)
	}
}

// Timing reports a timing value as a histogram in seconds.
func (m *VictoriaMetrics) Timing(name string, tags [][2]string) func(v time.Duration) {
	lbls := formatTags(tags, m.fqn)
	key := createKey(name, lbls, m.fqn)

	h := m.set.GetOrCreateHistogram(key)

	return func(v time.Duration) {
		h.Update(float64(v) / float64(time.Second))
	}
}

// Close closes the client and flushes buffered stats, if applicable.
func (m *VictoriaMetrics) Close() error {
	return nil
}

// createKey creates a unique metric key.
func createKey(name, lbls string, fqn *fqn) string {
	if lbls == "" {
		return fqn.Format(name)
	}
	return fqn.Format(name) + "{" + lbls + "}"
}

var pool = bytes.NewPool(512)

// formatTags create a prometheus Label map from tags.
func formatTags(tags [][2]string, fqn *fqn) string {
	if len(tags) == 0 {
		return ""
	}

	sort.Slice(tags, func(i, j int) bool {
		return tags[i][0] < tags[j][0]
	})

	buf := pool.Get()
	for i, tag := range tags {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(fqn.Format(tag[0]))
		buf.WriteByte('=')
		buf.WriteByte('"')
		buf.WriteString(tag[1])
		buf.WriteByte('"')
	}

	s := string(buf.Bytes())
	pool.Put(buf)
	return s
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
