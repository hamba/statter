package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hamba/statter/v2"
	"github.com/hamba/statter/v2/reporter/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestPrometheus_Handler(t *testing.T) {
	p := prometheus.New("test.test")
	t.Cleanup(func() { _ = p.Close() })

	h := p.Handler()

	assert.Implements(t, (*http.Handler)(nil), h)
}

func TestNew(t *testing.T) {
	p := prometheus.New("test.test")
	t.Cleanup(func() { _ = p.Close() })

	assert.Implements(t, (*statter.Reporter)(nil), p)
	assert.Implements(t, (*statter.HistogramReporter)(nil), p)
	assert.Implements(t, (*statter.TimingReporter)(nil), p)
}

func TestPrometheus_Counter(t *testing.T) {
	p := prometheus.New("test.test")
	t.Cleanup(func() { _ = p.Close() })

	p.Counter("test", 2, [][2]string{{"test", "test"}, {"foo", "bar"}})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.Contains(t, rr.Body.String(), "test_test_test{foo=\"bar\",test=\"test\"} 2")
}

func TestPrometheus_Gauge(t *testing.T) {
	p := prometheus.New("test.test")
	t.Cleanup(func() { _ = p.Close() })

	p.Gauge("test", 2.1, [][2]string{{"foo", "bar"}})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.Contains(t, rr.Body.String(), "test_test_test{foo=\"bar\"} 2.1")
}

func TestPrometheus_Histogram(t *testing.T) {
	p := prometheus.New("test.test", prometheus.WithBuckets([]float64{0.1, 1.0}))
	t.Cleanup(func() { _ = p.Close() })

	p.Histogram("test", [][2]string{{"foo", "bar"}})(0.0123)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.Contains(t, rr.Body.String(), "test_test_test_bucket{foo=\"bar\",le=\"0.1\"} 1")
	assert.Contains(t, rr.Body.String(), "test_test_test_bucket{foo=\"bar\",le=\"1\"} 1")
	assert.Contains(t, rr.Body.String(), "test_test_test_sum{foo=\"bar\"} 0.0123")
	assert.Contains(t, rr.Body.String(), "test_test_test_count{foo=\"bar\"} 1")
}

func TestPrometheus_Timing(t *testing.T) {
	p := prometheus.New("test.test", prometheus.WithBuckets([]float64{0.1, 1.0}))
	t.Cleanup(func() { _ = p.Close() })

	p.Timing("test", [][2]string{{"foo", "bar"}})(1234500 * time.Nanosecond)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.Contains(t, rr.Body.String(), "test_test_test_bucket{foo=\"bar\",le=\"0.1\"} 1")
	assert.Contains(t, rr.Body.String(), "test_test_test_bucket{foo=\"bar\",le=\"1\"} 1")
	assert.Contains(t, rr.Body.String(), "test_test_test_sum{foo=\"bar\"} 0.0012345")
	assert.Contains(t, rr.Body.String(), "test_test_test_count{foo=\"bar\"} 1")
}

func TestPrometheus_ConvertsLabels(t *testing.T) {
	p := prometheus.New("test.test")
	t.Cleanup(func() { _ = p.Close() })

	p.Counter("test", 2, [][2]string{{"test-label", "test"}})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.Contains(t, rr.Body.String(), "test_test_test{test_label=\"test\"} 2")
}

func TestPrometheus_NoPrefixNoTags(t *testing.T) {
	p := prometheus.New("")
	t.Cleanup(func() { _ = p.Close() })

	p.Counter("test", 2, [][2]string{})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.Contains(t, rr.Body.String(), "test 2")
}

func TestPrometheus_Close(t *testing.T) {
	p := prometheus.New("test.test")
	t.Cleanup(func() { _ = p.Close() })

	err := p.Close()

	assert.NoError(t, err)
}
