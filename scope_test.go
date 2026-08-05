package statter_test

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/hamba/statter/v2"
	"github.com/hamba/statter/v2/tags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	scopeID    = "scope"
	revKey     = "rev"
	revOld     = "abc"
	revNew     = "def"
	baseKey    = "base"
	baseVal    = "val"
	tagKey     = "tag"
	tagVal     = "tag-val"
	metricName = "test"
)

func TestStatter_Scope(t *testing.T) {
	m := &mockSimpleReporter{}
	m.On("Counter", "prefix.test", int64(2), [][2]string{{baseKey, baseVal}, {revKey, revOld}, {tagKey, tagVal}})
	m.On("Gauge", "prefix.test", 1.23, [][2]string{{baseKey, baseVal}, {revKey, revOld}})

	stats := statter.New(m, time.Second)

	sc := stats.With("prefix", tags.Str(baseKey, baseVal)).Scope(scopeID, tags.Str(revKey, revOld))
	sc.Counter(metricName, tags.Str(tagKey, tagVal)).Inc(2)
	sc.Gauge(metricName).Set(1.23)

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_ScopeReturnsIdenticalScope(t *testing.T) {
	stats := statter.New(statter.DiscardReporter, time.Second)
	t.Cleanup(func() { _ = stats.Close() })

	sc1 := stats.Scope(scopeID, tags.Str(revKey, revOld))
	sc2 := stats.Scope(scopeID, tags.Str(revKey, revOld))

	assert.Same(t, sc1, sc2)
}

func TestStatter_ScopeIsNotSharedBetweenStatters(t *testing.T) {
	stats := statter.New(statter.DiscardReporter, time.Second)
	t.Cleanup(func() { _ = stats.Close() })

	sc1 := stats.Scope(scopeID, tags.Str(revKey, revOld))
	sc2 := stats.With("prefix").Scope(scopeID, tags.Str(revKey, revOld))

	assert.NotSame(t, sc1, sc2)
}

func TestStatter_ScopeRotatesOnChangedTags(t *testing.T) {
	stats := statter.New(statter.DiscardReporter, time.Second)
	t.Cleanup(func() { _ = stats.Close() })

	sc1 := stats.Scope(scopeID, tags.Str(revKey, revOld))
	sc1.Gauge(metricName).Set(1)
	require.True(t, stats.HasGauge(metricName, tags.Str(revKey, revOld)))

	sc2 := stats.Scope(scopeID, tags.Str(revKey, revNew))
	sc2.Gauge(metricName).Set(1)

	assert.NotSame(t, sc1, sc2)
	assert.False(t, stats.HasGauge(metricName, tags.Str(revKey, revOld)), "old series should be removed")
	assert.True(t, stats.HasGauge(metricName, tags.Str(revKey, revNew)))
}

func TestStatter_ScopeRotateCallsRemoveOnReporter(t *testing.T) {
	m := &mockComplexReporter{}
	m.On("RemoveGauge", metricName, [][2]string{{revKey, revOld}})
	m.On("RemoveCounter", metricName, [][2]string{{revKey, revOld}})

	stats := statter.New(m, time.Second)

	sc := stats.Scope(scopeID, tags.Str(revKey, revOld))
	sc.Gauge(metricName).Set(1)
	sc.Counter(metricName).Inc(1)

	stats.Scope(scopeID, tags.Str(revKey, revNew))

	m.AssertExpectations(t)

	err := stats.Close()
	require.NoError(t, err)
}

func TestStatter_ScopeRotateWithSimpleReporter(t *testing.T) {
	m := &mockSimpleReporter{}

	stats := statter.New(m, time.Second)

	stats.Scope(scopeID, tags.Str(revKey, revOld)).Gauge(metricName).Set(1)
	stats.Scope(scopeID, tags.Str(revKey, revNew))

	err := stats.Close()
	require.NoError(t, err)

	assert.False(t, stats.HasGauge(metricName, tags.Str(revKey, revOld)))
	m.AssertExpectations(t)
}

func TestStatter_ScopeDelete(t *testing.T) {
	stats := statter.New(statter.DiscardReporter, time.Second)
	t.Cleanup(func() { _ = stats.Close() })

	sc := stats.Scope(scopeID, tags.Str(revKey, revOld))
	sc.Counter("counter").Inc(1)
	sc.Gauge("gauge").Set(1)
	sc.Histogram("histo").Observe(1)
	sc.Timing("timing").Observe(time.Second)

	sc.Delete()

	scopeTag := tags.Str(revKey, revOld)
	assert.False(t, stats.HasCounter("counter", scopeTag))
	assert.False(t, stats.HasGauge("gauge", scopeTag))
	assert.False(t, stats.HasHistogram("histo", scopeTag))
	assert.False(t, stats.HasTiming("timing", scopeTag))
}

func TestStatter_ScopeDeleteIsIdempotent(t *testing.T) {
	m := &mockComplexReporter{}
	m.On("RemoveGauge", metricName, [][2]string{{revKey, revOld}}).Once()

	stats := statter.New(m, time.Second)

	sc := stats.Scope(scopeID, tags.Str(revKey, revOld))
	sc.Gauge(metricName).Set(1)

	sc.Delete()
	sc.Delete()

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_ScopeDeleteUnregistersBeforeReporter(t *testing.T) {
	m := &mockComplexReporter{}

	stats := statter.New(m, time.Second)

	var got bool
	m.On("RemoveGauge", metricName, [][2]string{{revKey, revOld}}).Run(func(_ mock.Arguments) {
		got = stats.HasGauge(metricName, tags.Str(revKey, revOld))
	})

	sc := stats.Scope(scopeID, tags.Str(revKey, revOld))
	sc.Gauge(metricName).Set(1)

	sc.Delete()

	err := stats.Close()
	require.NoError(t, err)

	assert.False(t, got, "gauge should be unregistered before the reporter is called")
	m.AssertExpectations(t)
}

func TestStatter_ScopeDeleteRemovesAggregatedMetrics(t *testing.T) {
	m := &mockRemovableReporter{}
	m.On("RemoveCounter", "test_count", [][2]string{{revKey, revOld}})
	for _, name := range []string{"sum", "mean", "stddev", "min", "max", "10p", "90p"} {
		m.On("RemoveGauge", "test_"+name, [][2]string{{revKey, revOld}})
		m.On("RemoveGauge", "test_"+name+"_ms", [][2]string{{revKey, revOld}})
	}

	stats := statter.New(m, time.Second)

	sc := stats.Scope(scopeID, tags.Str(revKey, revOld))
	sc.Histogram(metricName).Observe(1)
	sc.Timing(metricName).Observe(time.Second)

	sc.Delete()

	m.AssertExpectations(t)
}

func TestStatter_ScopeStaleHandleDoesNotLeak(t *testing.T) {
	stats := statter.New(statter.DiscardReporter, time.Second)
	t.Cleanup(func() { _ = stats.Close() })

	sc := stats.Scope(scopeID, tags.Str(revKey, revOld))
	sc.Delete()

	sc.Gauge(metricName).Set(1)

	assert.False(t, stats.HasGauge(metricName, tags.Str(revKey, revOld)))
}

func TestStatter_ScopeFullName(t *testing.T) {
	stats := statter.New(statter.DiscardReporter, time.Second)
	t.Cleanup(func() { _ = stats.Close() })

	sc := stats.With("prefix").Scope(scopeID, tags.Str(revKey, revOld))

	assert.Equal(t, "prefix.test", sc.FullName(metricName))
}

func TestStatter_HasScope(t *testing.T) {
	stats := statter.New(statter.DiscardReporter, time.Second)
	t.Cleanup(func() { _ = stats.Close() })

	assert.False(t, stats.HasScope(scopeID))

	sc := stats.Scope(scopeID, tags.Str(revKey, revOld))
	assert.True(t, stats.HasScope(scopeID))

	sc.Delete()
	assert.False(t, stats.HasScope(scopeID))
}

func TestStatter_HasScopeAfterRotate(t *testing.T) {
	stats := statter.New(statter.DiscardReporter, time.Second)
	t.Cleanup(func() { _ = stats.Close() })

	sc := stats.Scope(scopeID, tags.Str(revKey, revOld))
	stats.Scope(scopeID, tags.Str(revKey, revNew))

	sc.Delete()

	assert.True(t, stats.HasScope(scopeID), "a stale scope must not remove its replacement")
}

func TestStatter_ScopeConcurrentRotate(t *testing.T) {
	stats := statter.New(statter.DiscardReporter, time.Second)
	t.Cleanup(func() { _ = stats.Close() })

	const n = 50

	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func() {
			defer wg.Done()

			rev := strconv.Itoa(i)
			stats.Scope(scopeID, tags.Str(revKey, rev)).Gauge(metricName).Set(1)
		}()
	}
	wg.Wait()

	var got int
	for i := range n {
		if stats.HasGauge(metricName, tags.Str(revKey, strconv.Itoa(i))) {
			got++
		}
	}

	assert.LessOrEqual(t, got, 1, "at most the surviving scope should have a gauge")
	assert.True(t, stats.HasScope(scopeID))
}

func TestStatter_ScopeConcurrentRotateWhileReporting(t *testing.T) {
	stats := statter.New(statter.DiscardReporter, time.Millisecond)
	t.Cleanup(func() { _ = stats.Close() })

	done := time.After(100 * time.Millisecond)
	for i := 0; ; i++ {
		select {
		case <-done:
			return
		default:
		}

		sc := stats.Scope(scopeID, tags.Str(revKey, strconv.Itoa(i)))
		sc.Gauge(metricName).Set(1)
		sc.Counter(metricName).Inc(1)
	}
}

func TestStatter_StaleMetricDeleteKeepsLiveMetric(t *testing.T) {
	stats := statter.New(statter.DiscardReporter, time.Second)
	t.Cleanup(func() { _ = stats.Close() })

	stale := stats.Gauge(metricName, tags.Str(tagKey, tagVal))
	stale.Delete()

	live := stats.Gauge(metricName, tags.Str(tagKey, tagVal))
	require.NotSame(t, stale, live)

	stale.Delete()

	assert.True(t, stats.HasGauge(metricName, tags.Str(tagKey, tagVal)), "the live gauge must survive")
}
