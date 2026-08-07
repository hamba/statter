package statter

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithPrefix(t *testing.T) {
	cfg := defaultConfig()

	WithPrefix("test-prefix")(&cfg)

	assert.Equal(t, "test-prefix", cfg.prefix)
}

func TestWithTags(t *testing.T) {
	cfg := defaultConfig()

	WithTags([2]string{"foo", "bar"}, [2]string{"baz", "bat"})(&cfg)

	assert.Equal(t, []Tag{[2]string{"foo", "bar"}, [2]string{"baz", "bat"}}, cfg.tags)
}

func TestWithSeparator(t *testing.T) {
	cfg := defaultConfig()

	WithSeparator("-")(&cfg)

	assert.Equal(t, "-", cfg.separator)
}

func TestWithPercentileSamples(t *testing.T) {
	cfg := defaultConfig()

	WithPercentileSamples(2)(&cfg)

	assert.Equal(t, 2, cfg.percSamples)
}

func TestWithPercentiles(t *testing.T) {
	cfg := defaultConfig()

	WithPercentiles([]float64{1, 2, 3})(&cfg)

	assert.Equal(t, []float64{1, 2, 3}, cfg.percentiles)
}

func TestStatter_ScopeDoesNotPolluteStatters(t *testing.T) {
	stats := New(DiscardReporter, time.Second)
	t.Cleanup(func() { _ = stats.Close() })

	stats.reg.mu.RLock()
	want := len(stats.reg.statters)
	stats.reg.mu.RUnlock()

	for i := range 100 {
		stats.Scope("scope", [2]string{"rev", strconv.Itoa(i)}).Gauge("test").Set(1)
	}

	stats.reg.mu.RLock()
	got := len(stats.reg.statters)
	stats.reg.mu.RUnlock()

	assert.Equal(t, want, got, "scopes must not be added to the statter cache")
}
