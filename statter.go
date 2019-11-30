package statter

import (
	"io"
	"time"
)

// Statter represents a stats instance.
type Statter interface {
	io.Closer

	// Inc increments a count by the value.
	Inc(name string, value int64, rate float32, tags ...string)

	// Gauge measures the value of a metric.
	Gauge(name string, value float64, rate float32, tags ...string)

	// Timing sends the value of a Duration.
	Timing(name string, value time.Duration, rate float32, tags ...string)
}
