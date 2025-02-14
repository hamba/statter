package victoriametrics_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hamba/statter/v2"
	"github.com/hamba/statter/v2/reporter/victoriametrics"
	"github.com/stretchr/testify/assert"
)

func TestVictoriaMetrics_Handler(t *testing.T) {
	p := victoriametrics.New()
	t.Cleanup(func() { _ = p.Close() })

	h := p.Handler()

	assert.Implements(t, (*http.Handler)(nil), h)
}

func TestNew(t *testing.T) {
	p := victoriametrics.New()
	t.Cleanup(func() { _ = p.Close() })

	assert.Implements(t, (*statter.Reporter)(nil), p)
	assert.Implements(t, (*statter.RemovableReporter)(nil), p)
	assert.Implements(t, (*statter.HistogramReporter)(nil), p)
	assert.Implements(t, (*statter.RemovableHistogramReporter)(nil), p)
	assert.Implements(t, (*statter.TimingReporter)(nil), p)
	assert.Implements(t, (*statter.RemovableTimingReporter)(nil), p)
}

func TestVictoriaMetrics_Counter(t *testing.T) {
	p := victoriametrics.New()
	t.Cleanup(func() { _ = p.Close() })

	p.Counter("test.test.test", 2, [][2]string{{"test", "test"}, {"foo", "bar"}})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.Contains(t, rr.Body.String(), "test_test_test{foo=\"bar\",test=\"test\"} 2")
}

func TestVictoriaMetrics_RemoveCounter(t *testing.T) {
	p := victoriametrics.New()
	t.Cleanup(func() { _ = p.Close() })

	p.Counter("test.test.test", 2, [][2]string{{"test", "test"}, {"foo", "bar"}})

	p.RemoveCounter("test.test.test", [][2]string{{"test", "test"}, {"foo", "bar"}})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.NotContains(t, rr.Body.String(), "test_test_test{foo=\"bar\",test=\"test\"} 2")
}

func TestVictoriaMetrics_Gauge(t *testing.T) {
	p := victoriametrics.New()
	t.Cleanup(func() { _ = p.Close() })

	p.Gauge("test.test.test", 2.1, [][2]string{{"foo", "bar"}})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.Contains(t, rr.Body.String(), "test_test_test{foo=\"bar\"} 2.1")
}

func TestVictoriaMetrics_RemoveGauge(t *testing.T) {
	p := victoriametrics.New()
	t.Cleanup(func() { _ = p.Close() })

	p.Gauge("test.test.test", 2.1, [][2]string{{"foo", "bar"}})

	p.RemoveGauge("test.test.test", [][2]string{{"foo", "bar"}})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.NotContains(t, rr.Body.String(), "test_test_test{foo=\"bar\"} 2.1")
}

func TestVictoriaMetrics_Histogram(t *testing.T) {
	p := victoriametrics.New()
	t.Cleanup(func() { _ = p.Close() })

	p.Histogram("test.test.test", [][2]string{{"foo", "bar"}})(0.0123)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.Contains(t, rr.Body.String(), "test_test_test_bucket{foo=\"bar\",vmrange=\"1.136e-02...1.292e-02\"} 1")
	assert.Contains(t, rr.Body.String(), "test_test_test_sum{foo=\"bar\"} 0.0123")
	assert.Contains(t, rr.Body.String(), "test_test_test_count{foo=\"bar\"} 1")
}

func TestVictoriaMetrics_RemoveHistogram(t *testing.T) {
	p := victoriametrics.New()
	t.Cleanup(func() { _ = p.Close() })

	p.Histogram("test.test.test", [][2]string{{"foo", "bar"}})(0.0123)

	p.RemoveHistogram("test.test.test", [][2]string{{"foo", "bar"}})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.NotContains(t, rr.Body.String(), "test_test_test_bucket{foo=\"bar\",vmrange=\"1.136e-02...1.292e-02\"} 1")
	assert.NotContains(t, rr.Body.String(), "test_test_test_sum{foo=\"bar\"} 0.0123")
	assert.NotContains(t, rr.Body.String(), "test_test_test_count{foo=\"bar\"} 1")
}

func TestVictoriaMetrics_Timing(t *testing.T) {
	p := victoriametrics.New()
	t.Cleanup(func() { _ = p.Close() })

	p.Timing("test.test.test", [][2]string{{"foo", "bar"}})(1234500 * time.Nanosecond)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.Contains(t, rr.Body.String(), "test_test_test_bucket{foo=\"bar\",vmrange=\"1.136e-03...1.292e-03\"} 1")
	assert.Contains(t, rr.Body.String(), "test_test_test_sum{foo=\"bar\"} 0.0012345")
	assert.Contains(t, rr.Body.String(), "test_test_test_count{foo=\"bar\"} 1")
}

func TestVictoriaMetrics_RemoveTiming(t *testing.T) {
	p := victoriametrics.New()
	t.Cleanup(func() { _ = p.Close() })

	p.Timing("test.test.test", [][2]string{{"foo", "bar"}})(1234500 * time.Nanosecond)

	p.RemoveTiming("test.test.test", [][2]string{{"foo", "bar"}})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.NotContains(t, rr.Body.String(), "test_test_test_bucket{foo=\"bar\",vmrange=\"1.136e-03...1.292e-03\"} 1")
	assert.NotContains(t, rr.Body.String(), "test_test_test_sum{foo=\"bar\"} 0.0012345")
	assert.NotContains(t, rr.Body.String(), "test_test_test_count{foo=\"bar\"} 1")
}

func TestVictoriaMetrics_ConvertsLabels(t *testing.T) {
	p := victoriametrics.New()
	t.Cleanup(func() { _ = p.Close() })

	p.Counter("foo.bar.baz", 2, [][2]string{{"test-label", "test"}, {"a", "b"}})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.Contains(t, rr.Body.String(), "foo_bar_baz{a=\"b\",test_label=\"test\"} 2")
}

func TestVictoriaMetrics_NoTags(t *testing.T) {
	p := victoriametrics.New()
	t.Cleanup(func() { _ = p.Close() })

	p.Counter("test", 2, [][2]string{})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.Contains(t, rr.Body.String(), "test 2")
}

func TestVictoriaMetrics_Close(t *testing.T) {
	p := victoriametrics.New()
	t.Cleanup(func() { _ = p.Close() })

	err := p.Close()

	assert.NoError(t, err)
}
