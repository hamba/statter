// Package victoriametrics implements a VictoriaMetrics stats reporter.
package victoriametrics

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/hamba/statter/v2/internal/bytes"
)

// VictoriaMetrics is a victoria metrics stats reporter.
type VictoriaMetrics struct {
	fqn *fqn

	set *metrics.Set
}

// New returns a new victoria metrics reporter.
func New() *VictoriaMetrics {
	fqn := newFQN()

	return &VictoriaMetrics{
		fqn: fqn,
		set: metrics.NewSet(),
	}
}

// Handler returns the VictoriaMetrics HTTP handler for scraping metrics in Prometheus format.
func (m *VictoriaMetrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

// RemoveCounter removes a counter.
func (m *VictoriaMetrics) RemoveCounter(name string, tags [][2]string) {
	m.removeMetric(name, tags)
}

// Gauge reports a gauge value.
func (m *VictoriaMetrics) Gauge(name string, v float64, tags [][2]string) {
	lbls := formatTags(tags, m.fqn)
	key := createKey(name, lbls, m.fqn)

	g := m.set.GetOrCreateGauge(key, nil)

	g.Set(v)
}

// RemoveGauge removes a gauge.
func (m *VictoriaMetrics) RemoveGauge(name string, tags [][2]string) {
	m.removeMetric(name, tags)
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

// RemoveHistogram removes a histogram.
func (m *VictoriaMetrics) RemoveHistogram(name string, tags [][2]string) {
	m.removeMetric(name, tags)
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

// RemoveTiming removes a timing.
func (m *VictoriaMetrics) RemoveTiming(name string, tags [][2]string) {
	m.removeMetric(name, tags)
}

func (m *VictoriaMetrics) removeMetric(name string, tags [][2]string) {
	lbls := formatTags(tags, m.fqn)
	key := createKey(name, lbls, m.fqn)

	m.set.UnregisterMetric(key)
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

	// The tags are owned by the caller and may be read concurrently, so they
	// must never be sorted in place.
	if !slices.IsSortedFunc(tags, compareTags) {
		tags = slices.Clone(tags)
		slices.SortFunc(tags, compareTags)
	}

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

func compareTags(a, b [2]string) int {
	return strings.Compare(a[0], b[0])
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
