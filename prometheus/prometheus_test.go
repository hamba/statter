package prometheus_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hamba/statter/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestPrometheus_Handler(t *testing.T) {
	l := &testLogger{}
	s := prometheus.New("test.test", l)

	h := s.Handler()

	assert.Implements(t, (*http.Handler)(nil), h)
}

func TestPrometheus_Inc(t *testing.T) {
	l := &testLogger{}
	s := prometheus.New("test.test", l)

	s.Inc("test", 2, 1.0, "test", "test")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	s.Handler().ServeHTTP(rr, req)

	assert.Equal(t, "msg=", l.Render())
	assert.Contains(t, rr.Body.String(), "test_test_test{test=\"test\"} 2")
}

func TestPrometheus_Dec(t *testing.T) {
	l := &testLogger{}
	s := prometheus.New("test.test", l)

	s.Dec("test", 2, 1.0, "test", "test")

	assert.Equal(t, "msg=prometheus: decrement not supported", l.Render())
}

func TestPrometheus_Gauge(t *testing.T) {
	l := &testLogger{}
	s := prometheus.New("test.test", l)

	s.Gauge("test", 2.1, 1.0, "test", "test")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	s.Handler().ServeHTTP(rr, req)

	assert.Equal(t, "msg=", l.Render())
	assert.Contains(t, rr.Body.String(), "test_test_test{test=\"test\"} 2.1")
}

func TestPrometheus_Timing(t *testing.T) {
	l := &testLogger{}
	s := prometheus.New("test.test", l)

	s.Timing("test", 1234500*time.Nanosecond, 1.0, "test", "test")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	s.Handler().ServeHTTP(rr, req)

	assert.Equal(t, "msg=", l.Render())
	assert.Contains(t, rr.Body.String(), "test_test_test{test=\"test\",quantile=\"0.5\"} 1.234")
	assert.Contains(t, rr.Body.String(), "test_test_test{test=\"test\",quantile=\"0.9\"} 1.234")
	assert.Contains(t, rr.Body.String(), "test_test_test{test=\"test\",quantile=\"0.99\"} 1.234")
	assert.Contains(t, rr.Body.String(), "test_test_test_sum{test=\"test\"} 1.234")
	assert.Contains(t, rr.Body.String(), "test_test_test_count{test=\"test\"} 1")
}

func TestPrometheus_ConvertsLabels(t *testing.T) {
	l := &testLogger{}
	s := prometheus.New("test.test", l)

	s.Inc("test", 2, 1.0, "test-label", "test")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	s.Handler().ServeHTTP(rr, req)

	assert.Equal(t, "msg=", l.Render())
	assert.Contains(t, rr.Body.String(), "test_test_test{test_label=\"test\"} 2")
}

func TestPrometheus_Close(t *testing.T) {
	l := &testLogger{}
	s := prometheus.New("test.test", l)

	err := s.Close()

	assert.NoError(t, err)
}

type testLogger struct {
	msg string
	ctx []interface{}
}

func (l *testLogger) Error(msg string, ctx ...interface{}) {
	l.msg = msg
	l.ctx = ctx
}

func (l *testLogger) Render() string {
	var buf bytes.Buffer
	for i := 0; i < len(l.ctx); i += 2 {
		buf.WriteString(fmt.Sprintf("%v=%v ", l.ctx[i], l.ctx[i+1]))
	}

	return strings.Trim(fmt.Sprintf("msg=%s %s", l.msg, buf.String()), " ")
}
