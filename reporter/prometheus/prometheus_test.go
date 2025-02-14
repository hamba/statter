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
	assert.Implements(t, (*statter.RemovableReporter)(nil), p)
	assert.Implements(t, (*statter.HistogramReporter)(nil), p)
	assert.Implements(t, (*statter.RemovableHistogramReporter)(nil), p)
	assert.Implements(t, (*statter.TimingReporter)(nil), p)
	assert.Implements(t, (*statter.RemovableTimingReporter)(nil), p)
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

func TestPrometheus_RemoveCounter(t *testing.T) {
	p := prometheus.New("test.test")
	t.Cleanup(func() { _ = p.Close() })

	p.Counter("test", 2, [][2]string{{"test", "test"}, {"foo", "bar"}})

	p.RemoveCounter("test", [][2]string{{"test", "test"}, {"foo", "bar"}})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.NotContains(t, rr.Body.String(), "test_test_test{foo=\"bar\",test=\"test\"} 2")
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

func TestPrometheus_RemoveGauge(t *testing.T) {
	p := prometheus.New("test.test")
	t.Cleanup(func() { _ = p.Close() })

	p.Gauge("test", 2.1, [][2]string{{"foo", "bar"}})

	p.RemoveGauge("test", [][2]string{{"foo", "bar"}})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.NotContains(t, rr.Body.String(), "test_test_test{foo=\"bar\"} 2.1")
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

func TestPrometheus_RemoveHistogram(t *testing.T) {
	p := prometheus.New("test.test", prometheus.WithBuckets([]float64{0.1, 1.0}))
	t.Cleanup(func() { _ = p.Close() })

	p.Histogram("test", [][2]string{{"foo", "bar"}})(0.0123)

	p.RemoveHistogram("test", [][2]string{{"foo", "bar"}})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.NotContains(t, rr.Body.String(), "test_test_test_bucket{foo=\"bar\",le=\"0.1\"} 1")
	assert.NotContains(t, rr.Body.String(), "test_test_test_bucket{foo=\"bar\",le=\"1\"} 1")
	assert.NotContains(t, rr.Body.String(), "test_test_test_sum{foo=\"bar\"} 0.0123")
	assert.NotContains(t, rr.Body.String(), "test_test_test_count{foo=\"bar\"} 1")
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

func TestPrometheus_RemoveTiming(t *testing.T) {
	p := prometheus.New("test.test", prometheus.WithBuckets([]float64{0.1, 1.0}))
	t.Cleanup(func() { _ = p.Close() })

	p.Timing("test", [][2]string{{"foo", "bar"}})(1234500 * time.Nanosecond)

	p.RemoveTiming("test", [][2]string{{"foo", "bar"}})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.NotContains(t, rr.Body.String(), "test_test_test_bucket{foo=\"bar\",le=\"0.1\"} 1")
	assert.NotContains(t, rr.Body.String(), "test_test_test_bucket{foo=\"bar\",le=\"1\"} 1")
	assert.NotContains(t, rr.Body.String(), "test_test_test_sum{foo=\"bar\"} 0.0012345")
	assert.NotContains(t, rr.Body.String(), "test_test_test_count{foo=\"bar\"} 1")
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

func TestRegisterCounter(t *testing.T) {
	p := prometheus.New("foo.bar")
	stats := statter.New(p, time.Second).With("baz")

	prometheus.RegisterCounter(stats, "bat", []string{"label1"}, "my awesome counter")

	p.Counter("baz.bat", 1, [][2]string{{"label1", "value1"}})

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	p.Handler().ServeHTTP(rec, req)

	assert.Contains(t, rec.Body.String(), `HELP foo_bar_baz_bat my awesome counter`)
	assert.Contains(t, rec.Body.String(), `foo_bar_baz_bat{label1="value1"} 1`)
}

func TestRegisterGauge(t *testing.T) {
	p := prometheus.New("foo.bar")
	stats := statter.New(p, time.Second).With("baz")

	prometheus.RegisterGauge(stats, "bat", []string{"label1"}, "my awesome gauge")

	p.Gauge("baz.bat", 1.23, [][2]string{{"label1", "value1"}})

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	p.Handler().ServeHTTP(rec, req)

	assert.Contains(t, rec.Body.String(), `HELP foo_bar_baz_bat my awesome gauge`)
	assert.Contains(t, rec.Body.String(), `foo_bar_baz_bat{label1="value1"} 1.23`)
}

func TestRegisterHistogram(t *testing.T) {
	p := prometheus.New("foo.bar")
	stats := statter.New(p, time.Second).With("baz")

	prometheus.RegisterHistogram(stats, "bat", []string{"label1"}, []float64{0.1, 1.0}, "my awesome histogram")

	p.Histogram("baz.bat", [][2]string{{"label1", "value1"}})(0.0123)

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	p.Handler().ServeHTTP(rec, req)

	assert.Contains(t, rec.Body.String(), `HELP foo_bar_baz_bat my awesome histogram`)
	assert.Contains(t, rec.Body.String(), "foo_bar_baz_bat_bucket{label1=\"value1\",le=\"0.1\"} 1")
	assert.Contains(t, rec.Body.String(), "foo_bar_baz_bat_bucket{label1=\"value1\",le=\"1\"} 1")
	assert.Contains(t, rec.Body.String(), "foo_bar_baz_bat_sum{label1=\"value1\"} 0.0123")
	assert.Contains(t, rec.Body.String(), "foo_bar_baz_bat_count{label1=\"value1\"} 1")
}

func TestRegisterHistogram_HandlesNoBuckets(t *testing.T) {
	p := prometheus.New("foo.bar")
	stats := statter.New(p, time.Second).With("baz")

	prometheus.RegisterHistogram(stats, "bat", []string{"label1"}, nil, "my awesome histogram")

	p.Histogram("baz.bat", [][2]string{{"label1", "value1"}})(0.0123)

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	p.Handler().ServeHTTP(rec, req)

	assert.Contains(t, rec.Body.String(), `HELP foo_bar_baz_bat my awesome histogram`)
	assert.Contains(t, rec.Body.String(), "foo_bar_baz_bat_bucket{label1=\"value1\",le=\"0.1\"} 1")
	assert.Contains(t, rec.Body.String(), "foo_bar_baz_bat_bucket{label1=\"value1\",le=\"0.025\"} 1")
	assert.Contains(t, rec.Body.String(), "foo_bar_baz_bat_bucket{label1=\"value1\",le=\"1\"} 1")
	assert.Contains(t, rec.Body.String(), "foo_bar_baz_bat_sum{label1=\"value1\"} 0.0123")
	assert.Contains(t, rec.Body.String(), "foo_bar_baz_bat_count{label1=\"value1\"} 1")
}

func TestSetBuckets_Histogram(t *testing.T) {
	p := prometheus.New("test.test")
	stats := statter.New(p, time.Second)

	prometheus.SetMetricBuckets(stats, "test", []float64{0.1, 1.0})

	p.Histogram("test", [][2]string{{"foo", "bar"}})(0.0123)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.Contains(t, rr.Body.String(), "test_test_test_bucket{foo=\"bar\",le=\"0.1\"} 1")
	assert.Contains(t, rr.Body.String(), "test_test_test_bucket{foo=\"bar\",le=\"1\"} 1")
	assert.Contains(t, rr.Body.String(), "test_test_test_sum{foo=\"bar\"} 0.0123")
	assert.Contains(t, rr.Body.String(), "test_test_test_count{foo=\"bar\"} 1")
}

func TestSetBuckets_Timing(t *testing.T) {
	p := prometheus.New("test.test")
	stats := statter.New(p, time.Second)

	prometheus.SetMetricBuckets(stats, "test", []float64{0.1, 1.0})

	p.Timing("test", [][2]string{{"foo", "bar"}})(1234500 * time.Nanosecond)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.Handler().ServeHTTP(rr, req)

	assert.Contains(t, rr.Body.String(), "test_test_test_bucket{foo=\"bar\",le=\"0.1\"} 1")
	assert.Contains(t, rr.Body.String(), "test_test_test_bucket{foo=\"bar\",le=\"1\"} 1")
	assert.Contains(t, rr.Body.String(), "test_test_test_sum{foo=\"bar\"} 0.0012345")
	assert.Contains(t, rr.Body.String(), "test_test_test_count{foo=\"bar\"} 1")
}
