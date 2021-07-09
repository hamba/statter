package statter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
