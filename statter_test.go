package statter_test

import (
	"sync"
	"testing"
	"time"

	"github.com/hamba/statter/v2"
	"github.com/hamba/statter/v2/tags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNew_HandlesOptions(t *testing.T) {
	m := &mockSimpleReporter{}
	m.On("Counter", "test-test", int64(2), [][2]string{{"tag", "test"}})

	stats := statter.New(m, time.Second, statter.WithSeparator("-"))

	stats.With("test").Counter("test", tags.Str("tag", "test")).Inc(2)

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_CallsReporter(t *testing.T) {
	m := newWaitingReporter()
	m.On("Counter", "test", int64(2), [][2]string{{"tag", "test"}})

	stats := statter.New(m, time.Millisecond)
	t.Cleanup(func() { _ = stats.Close() })

	stats.Counter("test", tags.Str("tag", "test")).Inc(2)

	select {
	case <-time.After(time.Second):
		assert.FailNow(t, "expected call to reporter timed out")
	case <-m.Ch():
	}

	m.AssertExpectations(t)
}

func TestStatter_With(t *testing.T) {
	m := &mockSimpleReporter{}
	m.On("Counter", "prefix.prefix2.test", int64(2), [][2]string{{"base", "val"}, {"base2", "val2"}, {"tag", "test"}})

	stats := statter.New(m, time.Second)

	stats.With("prefix", tags.Str("base", "val")).
		With("", tags.Str("base2", "val2")).
		With("prefix2").
		Counter("test", tags.Str("tag", "test")).Inc(2)

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_WithReturnsTheRootStatterWithEmpty(t *testing.T) {
	m := &mockSimpleReporter{}

	stats := statter.New(m, time.Second, statter.WithPrefix("prefix"), statter.WithTags(tags.Str("base", "val")))
	t.Cleanup(func() { _ = stats.Close() })

	got := stats.With("")

	assert.Same(t, stats, got)
	m.AssertExpectations(t)
}

func TestStatter_WithReturnsIdenticalStatter(t *testing.T) {
	stats := statter.New(statter.DiscardReporter, time.Second)
	t.Cleanup(func() { _ = stats.Close() })

	s1 := stats.With("test", tags.Str("tag", "test"))

	s2 := stats.With("test", tags.Str("tag", "test"))

	assert.Same(t, s1, s2)
}

func TestStatter_Counter(t *testing.T) {
	m := &mockSimpleReporter{}
	m.On("Counter", "test", int64(2), [][2]string{{"tag", "test"}})

	stats := statter.New(m, time.Second)

	stats.Counter("test", tags.Str("tag", "test")).Inc(2)

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_CounterReturnsIdenticalCounter(t *testing.T) {
	stats := statter.New(statter.DiscardReporter, time.Second)
	t.Cleanup(func() { _ = stats.Close() })

	c1 := stats.Counter("test", tags.Str("tag", "test"))

	c2 := stats.Counter("test", tags.Str("tag", "test"))

	assert.Same(t, c1, c2)
}

func TestStatter_HasCounter(t *testing.T) {
	m := &mockSimpleReporter{}
	m.On("Counter", "test", int64(2), [][2]string{{"tag", "test"}})

	stats := statter.New(m, time.Second)
	t.Cleanup(func() { _ = stats.Close() })
	stats.Counter("test", tags.Str("tag", "test")).Inc(2)

	gotExists := stats.HasCounter("test", tags.Str("tag", "test"))
	gotNotExists := stats.HasCounter("other", tags.Str("tag", "test"))
	gotNoTag := stats.HasCounter("test", tags.Str("other", "test"))

	assert.True(t, gotExists)
	assert.False(t, gotNotExists)
	assert.False(t, gotNoTag)
}

func TestStatter_CounterDelete(t *testing.T) {
	m := &mockSimpleReporter{}

	stats := statter.New(m, time.Second)
	stats.Counter("test", tags.Str("tag", "test")).Inc(2)

	stats.Counter("test", tags.Str("tag", "test")).Delete()

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_CounterComplexDelete(t *testing.T) {
	m := &mockComplexReporter{}
	m.On("RemoveCounter", "test", [][2]string{{"tag", "test"}})

	stats := statter.New(m, time.Second)
	stats.Counter("test", tags.Str("tag", "test")).Inc(2)

	stats.Counter("test", tags.Str("tag", "test")).Delete()

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_Gauge(t *testing.T) {
	m := &mockSimpleReporter{}
	m.On("Gauge", "test", 1.23, [][2]string{{"tag", "test"}})

	stats := statter.New(m, time.Second)

	stats.Gauge("test", tags.Str("tag", "test")).Set(1.23)

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_GaugeAddSub(t *testing.T) {
	m := &mockSimpleReporter{}
	m.On("Gauge", "test", 0.0, [][2]string{{"tag", "test"}})

	stats := statter.New(m, time.Second)

	stats.Gauge("test", tags.Str("tag", "test")).Add(1.23)
	stats.Gauge("test", tags.Str("tag", "test")).Sub(1.23)

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_GaugeIncDec(t *testing.T) {
	m := &mockSimpleReporter{}
	m.On("Gauge", "test", 0.0, [][2]string{{"tag", "test"}})

	stats := statter.New(m, time.Second)

	stats.Gauge("test", tags.Str("tag", "test")).Inc()
	stats.Gauge("test", tags.Str("tag", "test")).Dec()

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_GaugeReturnsIdenticalCounter(t *testing.T) {
	stats := statter.New(statter.DiscardReporter, time.Second)
	t.Cleanup(func() { _ = stats.Close() })

	g1 := stats.Gauge("test", tags.Str("tag", "test"))

	g2 := stats.Gauge("test", tags.Str("tag", "test"))

	assert.Same(t, g1, g2)
}

func TestStatter_HasGauge(t *testing.T) {
	m := &mockSimpleReporter{}
	m.On("Gauge", "test", 1.23, [][2]string{{"tag", "test"}})

	stats := statter.New(m, time.Second)
	t.Cleanup(func() { _ = stats.Close() })
	stats.Gauge("test", tags.Str("tag", "test")).Set(1.23)

	gotExists := stats.HasGauge("test", tags.Str("tag", "test"))
	gotNotExists := stats.HasGauge("other", tags.Str("tag", "test"))
	gotNoTag := stats.HasGauge("test", tags.Str("other", "test"))

	assert.True(t, gotExists)
	assert.False(t, gotNotExists)
	assert.False(t, gotNoTag)
}

func TestStatter_GaugeDelete(t *testing.T) {
	m := &mockSimpleReporter{}

	stats := statter.New(m, time.Second)
	stats.Gauge("test", tags.Str("tag", "test")).Set(1.23)

	stats.Gauge("test", tags.Str("tag", "test")).Delete()

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_GaugeComplexDelete(t *testing.T) {
	m := &mockComplexReporter{}
	m.On("RemoveGauge", "test", [][2]string{{"tag", "test"}})

	stats := statter.New(m, time.Second)
	stats.Gauge("test", tags.Str("tag", "test")).Set(1.23)

	stats.Gauge("test", tags.Str("tag", "test")).Delete()

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_Histogram(t *testing.T) {
	m := &mockComplexReporter{}
	m.On("Histogram", "test", [][2]string{{"tag", "test"}}).Return(func(v float64) {
		assert.Equal(t, 10.0, v)
	})

	stats := statter.New(m, time.Second)

	stats.Histogram("test", tags.Str("tag", "test")).Observe(10)

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_HistogramAggregated(t *testing.T) {
	m := &mockSimpleReporter{}
	m.On("Counter", "test_count", int64(16), [][2]string{{"tag", "test"}}).Once()
	m.On("Gauge", "test_sum", 255.0, [][2]string{{"tag", "test"}}).Once()
	m.On("Gauge", "test_mean", 15.9375, [][2]string{{"tag", "test"}}).Once()
	m.On("Gauge", "test_stddev", 11.177369715187917, [][2]string{{"tag", "test"}}).Once()
	m.On("Gauge", "test_min", 5.0, [][2]string{{"tag", "test"}}).Once()
	m.On("Gauge", "test_max", 45.0, [][2]string{{"tag", "test"}}).Once()
	m.On("Gauge", "test_10p", 5.0, [][2]string{{"tag", "test"}}).Once()
	m.On("Gauge", "test_90p", 32.0, [][2]string{{"tag", "test"}}).Once()

	values := []float64{10, 20, 10, 30, 20, 11, 12, 32, 45, 9, 5, 5, 5, 10, 23, 8}

	stats := statter.New(m, time.Second)

	h := stats.Histogram("test", tags.Str("tag", "test"))
	for _, v := range values {
		h.Observe(v)
	}

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_HistogramAggregatedSwapsSamples(t *testing.T) {
	m := newWaitingReporter()
	m.On("Counter", "test_count", int64(1), [][2]string{{"tag", "test"}}).Twice()
	m.On("Gauge", "test_sum", 10.0, [][2]string{{"tag", "test"}}).Twice()
	m.On("Gauge", "test_mean", 10.0, [][2]string{{"tag", "test"}}).Twice()
	m.On("Gauge", "test_stddev", 0.0, [][2]string{{"tag", "test"}}).Twice()
	m.On("Gauge", "test_min", 10.0, [][2]string{{"tag", "test"}}).Twice()
	m.On("Gauge", "test_max", 10.0, [][2]string{{"tag", "test"}}).Twice()
	m.On("Gauge", "test_10p", 10.0, [][2]string{{"tag", "test"}}).Twice()
	m.On("Gauge", "test_90p", 10.0, [][2]string{{"tag", "test"}}).Twice()

	stats := statter.New(m, time.Millisecond)

	stats.Histogram("test", tags.Str("tag", "test")).Observe(10)

	select {
	case <-time.After(time.Second):
		assert.FailNow(t, "expected call to reporter timed out")
	case <-m.Ch():
	}
	m.Reset()

	stats.Histogram("test", tags.Str("tag", "test")).Observe(10)

	select {
	case <-time.After(time.Second):
		assert.FailNow(t, "expected call to reporter timed out")
	case <-m.Ch():
	}

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_HistogramReturnsIdenticalCounter(t *testing.T) {
	stats := statter.New(statter.DiscardReporter, time.Second)
	t.Cleanup(func() { _ = stats.Close() })

	h1 := stats.Histogram("test", tags.Str("tag", "test"))

	h2 := stats.Histogram("test", tags.Str("tag", "test"))

	assert.Same(t, h1, h2)
}

func TestStatter_HasHistogram(t *testing.T) {
	m := &mockComplexReporter{}
	m.On("Histogram", "test", [][2]string{{"tag", "test"}}).Return(func(float64) {})

	stats := statter.New(m, time.Second)
	t.Cleanup(func() { _ = stats.Close() })
	stats.Histogram("test", tags.Str("tag", "test")).Observe(10)

	gotExists := stats.HasHistogram("test", tags.Str("tag", "test"))
	gotNotExists := stats.HasHistogram("other", tags.Str("tag", "test"))
	gotNoTag := stats.HasHistogram("test", tags.Str("other", "test"))

	assert.True(t, gotExists)
	assert.False(t, gotNotExists)
	assert.False(t, gotNoTag)
}

func TestStatter_HistogramDelete(t *testing.T) {
	m := &mockComplexReporter{}
	m.On("Histogram", "test", [][2]string{{"tag", "test"}}).Return(func(v float64) {
		assert.Equal(t, 10.0, v)
	})
	m.On("RemoveHistogram", "test", [][2]string{{"tag", "test"}})

	stats := statter.New(m, time.Second)
	stats.Histogram("test", tags.Str("tag", "test")).Observe(10)

	stats.Histogram("test", tags.Str("tag", "test")).Delete()

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_HistogramAggregatedDelete(t *testing.T) {
	m := &mockSimpleReporter{}

	values := []float64{10, 20, 10, 30, 20, 11, 12, 32, 45, 9, 5, 5, 5, 10, 23, 8}

	stats := statter.New(m, time.Second)

	h := stats.Histogram("test", tags.Str("tag", "test"))
	for _, v := range values {
		h.Observe(v)
	}

	h.Delete()

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_Timing(t *testing.T) {
	m := &mockComplexReporter{}
	m.On("Timing", "test", [][2]string{{"tag", "test"}}).Return(func(v time.Duration) {
		assert.Equal(t, 10*time.Millisecond, v)
	})

	stats := statter.New(m, time.Second)

	stats.Timing("test", tags.Str("tag", "test")).Observe(10 * time.Millisecond)

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_TimingAggregated(t *testing.T) {
	m := &mockSimpleReporter{}
	m.On("Counter", "test_count", int64(16), [][2]string{{"tag", "test"}}).Once()
	m.On("Gauge", "test_sum_ms", 255.0, [][2]string{{"tag", "test"}}).Once()
	m.On("Gauge", "test_mean_ms", 15.9375, [][2]string{{"tag", "test"}}).Once()
	m.On("Gauge", "test_stddev_ms", 11.177369715187917, [][2]string{{"tag", "test"}}).Once()
	m.On("Gauge", "test_min_ms", 5.0, [][2]string{{"tag", "test"}}).Once()
	m.On("Gauge", "test_max_ms", 45.0, [][2]string{{"tag", "test"}}).Once()
	m.On("Gauge", "test_10p_ms", 5.0, [][2]string{{"tag", "test"}}).Once()
	m.On("Gauge", "test_90p_ms", 32.0, [][2]string{{"tag", "test"}}).Once()

	values := []int{10, 20, 10, 30, 20, 11, 12, 32, 45, 9, 5, 5, 5, 10, 23, 8}

	stats := statter.New(m, time.Second)

	timing := stats.Timing("test", tags.Str("tag", "test"))
	for _, v := range values {
		timing.Observe(time.Duration(v) * time.Millisecond)
	}

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_TimingAggregatedSwapsSamples(t *testing.T) {
	m := newWaitingReporter()
	m.On("Counter", "test_count", int64(1), [][2]string{{"tag", "test"}}).Twice()
	m.On("Gauge", "test_sum_ms", 10.0, [][2]string{{"tag", "test"}}).Twice()
	m.On("Gauge", "test_mean_ms", 10.0, [][2]string{{"tag", "test"}}).Twice()
	m.On("Gauge", "test_stddev_ms", 0.0, [][2]string{{"tag", "test"}}).Twice()
	m.On("Gauge", "test_min_ms", 10.0, [][2]string{{"tag", "test"}}).Twice()
	m.On("Gauge", "test_max_ms", 10.0, [][2]string{{"tag", "test"}}).Twice()
	m.On("Gauge", "test_10p_ms", 10.0, [][2]string{{"tag", "test"}}).Twice()
	m.On("Gauge", "test_90p_ms", 10.0, [][2]string{{"tag", "test"}}).Twice()

	stats := statter.New(m, time.Millisecond)

	stats.Timing("test", tags.Str("tag", "test")).Observe(10 * time.Millisecond)

	select {
	case <-time.After(time.Second):
		assert.FailNow(t, "expected call to reporter timed out")
	case <-m.Ch():
	}
	m.Reset()

	stats.Timing("test", tags.Str("tag", "test")).Observe(10 * time.Millisecond)

	select {
	case <-time.After(time.Second):
		assert.FailNow(t, "expected call to reporter timed out")
	case <-m.Ch():
	}

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_AggregatedCallsNothingIfNoValues(t *testing.T) {
	m := &mockSimpleReporter{}
	m.Test(t)

	stats := statter.New(m, time.Second)

	err := stats.Close()
	require.NoError(t, err)

	m.AssertNotCalled(t, "Counter", mock.Anything)
	m.AssertNotCalled(t, "Gauge", mock.Anything)
}

func TestStatter_TimingReturnsIdenticalCounter(t *testing.T) {
	stats := statter.New(statter.DiscardReporter, time.Second)
	t.Cleanup(func() { _ = stats.Close() })

	t1 := stats.Timing("test", tags.Str("tag", "test"))

	t2 := stats.Timing("test", tags.Str("tag", "test"))

	assert.Same(t, t1, t2)
}

func TestStatter_HasTiming(t *testing.T) {
	m := &mockComplexReporter{}
	m.On("Timing", "test", [][2]string{{"tag", "test"}}).Return(func(time.Duration) {})

	stats := statter.New(m, time.Second)
	t.Cleanup(func() { _ = stats.Close() })
	stats.Timing("test", tags.Str("tag", "test")).Observe(10 * time.Millisecond)

	gotExists := stats.HasTiming("test", tags.Str("tag", "test"))
	gotNotExists := stats.HasTiming("other", tags.Str("tag", "test"))
	gotNoTag := stats.HasTiming("test", tags.Str("other", "test"))

	assert.True(t, gotExists)
	assert.False(t, gotNotExists)
	assert.False(t, gotNoTag)
}

func TestStatter_TimingDelete(t *testing.T) {
	m := &mockComplexReporter{}
	m.On("Timing", "test", [][2]string{{"tag", "test"}}).Return(func(v time.Duration) {
		assert.Equal(t, 10*time.Millisecond, v)
	})
	m.On("RemoveTiming", "test", [][2]string{{"tag", "test"}})

	stats := statter.New(m, time.Second)
	stats.Timing("test", tags.Str("tag", "test")).Observe(10 * time.Millisecond)

	stats.Timing("test", tags.Str("tag", "test")).Delete()

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_TimingAggregatedDelete(t *testing.T) {
	m := &mockSimpleReporter{}

	values := []int{10, 20, 10, 30, 20, 11, 12, 32, 45, 9, 5, 5, 5, 10, 23, 8}

	stats := statter.New(m, time.Second)

	timing := stats.Timing("test", tags.Str("tag", "test"))
	for _, v := range values {
		timing.Observe(time.Duration(v) * time.Millisecond)
	}

	timing.Delete()

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_CloseFromSubStatterFails(t *testing.T) {
	stats := statter.New(statter.DiscardReporter, time.Second).With("prefix", tags.Str("base", "val"))

	err := stats.Close()

	assert.Error(t, err)
}

func TestNullReporter(t *testing.T) {
	assert.Implements(t, (*statter.Reporter)(nil), statter.DiscardReporter)
	assert.Implements(t, (*statter.HistogramReporter)(nil), statter.DiscardReporter)
	assert.Implements(t, (*statter.TimingReporter)(nil), statter.DiscardReporter)
}

type mockSimpleReporter struct {
	mock.Mock
}

func (r *mockSimpleReporter) Counter(name string, v int64, tags [][2]string) {
	_ = r.Called(name, v, tags)
}

func (r *mockSimpleReporter) Gauge(name string, v float64, tags [][2]string) {
	_ = r.Called(name, v, tags)
}

func (r *mockSimpleReporter) Close() error {
	return nil
}

type mockComplexReporter struct {
	mock.Mock
}

func (r *mockComplexReporter) Counter(name string, v int64, tags [][2]string) {
	_ = r.Called(name, v, tags)
}

func (r *mockComplexReporter) RemoveCounter(name string, tags [][2]string) {
	_ = r.Called(name, tags)
}

func (r *mockComplexReporter) Gauge(name string, v float64, tags [][2]string) {
	_ = r.Called(name, v, tags)
}

func (r *mockComplexReporter) RemoveGauge(name string, tags [][2]string) {
	_ = r.Called(name, tags)
}

func (r *mockComplexReporter) Histogram(name string, tags [][2]string) func(v float64) {
	args := r.Called(name, tags)

	ret := args.Get(0)
	if ret == nil {
		return nil
	}
	return ret.(func(v float64))
}

func (r *mockComplexReporter) RemoveHistogram(name string, tags [][2]string) {
	_ = r.Called(name, tags)
}

func (r *mockComplexReporter) Timing(name string, tags [][2]string) func(v time.Duration) {
	args := r.Called(name, tags)

	ret := args.Get(0)
	if ret == nil {
		return nil
	}
	return ret.(func(v time.Duration))
}

func (r *mockComplexReporter) RemoveTiming(name string, tags [][2]string) {
	_ = r.Called(name, tags)
}

type waitingReporter struct {
	mock.Mock

	mu     sync.Mutex
	ch     chan struct{}
	closed bool
}

func newWaitingReporter() *waitingReporter {
	return &waitingReporter{
		ch: make(chan struct{}),
	}
}

func (r *waitingReporter) Counter(name string, v int64, tags [][2]string) {
	_ = r.Called(name, v, tags)

	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.closed {
		close(r.ch)
		r.closed = true
	}
}

func (r *waitingReporter) Gauge(name string, v float64, tags [][2]string) {
	_ = r.Called(name, v, tags)

	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.closed {
		close(r.ch)
		r.closed = true
	}
}

func (r *waitingReporter) Ch() <-chan struct{} {
	return r.ch
}

func (r *waitingReporter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ch = make(chan struct{})
	r.closed = false
}
