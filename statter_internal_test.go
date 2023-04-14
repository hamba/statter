package statter

import (
	"testing"

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
