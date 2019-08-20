// Package l2met implements an l2met stats client.
package l2met

import (
	"math/rand"
	"time"

	"github.com/hamba/statter/internal/bytes"
	"github.com/hamba/statter/internal/tags"
)

// Logger represents an abstract logging object.
type Logger interface {
	Info(msg string, ctx ...interface{})
}

// SamplerFunc represents a function that samples the L2met stats.
type SamplerFunc func(float32) bool

func defaultSampler(rate float32) bool {
	if rand.Float32() < rate {
		return true
	}

	return false
}

// OptsFunc represents a function that configures L2met.
type OptsFunc func(*L2met)

// UseRates turns on sample rates in l2met.
func UseRates() OptsFunc {
	return func(s *L2met) {
		s.useRates = true
	}
}

// UseSampler sets the sampler function for l2met.
func UseSampler(sampler SamplerFunc) OptsFunc {
	return func(s *L2met) {
		s.sampler = sampler
	}
}

// L2met is a l2met client.
type L2met struct {
	log    Logger
	prefix string

	useRates bool
	sampler  SamplerFunc
}

// New returns a l2met instance.
func New(l Logger, prefix string, opts ...OptsFunc) *L2met {
	if len(prefix) > 0 {
		prefix += "."
	}

	s := &L2met{
		log:     l,
		prefix:  prefix,
		sampler: defaultSampler,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Inc increments a count by the value.
func (s *L2met) Inc(name string, value int64, rate float32, tags ...string) {
	s.render(
		"count",
		name,
		value,
		rate,
		tags,
	)
}

// Gauge measures the value of a metric.
func (s *L2met) Gauge(name string, value float64, rate float32, tags ...string) {
	s.render(
		"sample",
		name,
		value,
		rate,
		tags,
	)
}

// Timing sends the value of a Duration.
func (s *L2met) Timing(name string, value time.Duration, rate float32, tags ...string) {
	s.render(
		"measure",
		name,
		formatDuration(value),
		rate,
		tags,
	)
}

// render outputs the metric to the logger
func (s *L2met) render(measure, name string, value interface{}, rate float32, t []string) {
	if !s.includeStat(rate) {
		return
	}

	t = tags.Deduplicate(tags.Normalize(t))

	ctx := make([]interface{}, len(t)+2)
	ctx[0] = measure + "#" + s.prefix + name + s.formatRate(rate)
	ctx[1] = value
	for i, val := range t {
		ctx[i+2] = val
	}

	s.log.Info("", ctx...)
}

func (s *L2met) includeStat(rate float32) bool {
	if !s.useRates || rate == 1.0 {
		return true
	}

	return s.sampler(rate)
}

// Close closes the client and flushes buffered stats, if applicable
func (s *L2met) Close() error {
	return nil
}

var pool = bytes.NewPool(512)

// formatRate creates an l2met compatible rate suffix.
func (s *L2met) formatRate(rate float32) string {
	if !s.useRates || rate == 1.0 {
		return ""
	}

	buf := pool.Get()
	buf.WriteByte('@')
	buf.AppendFloat(float64(rate), 'f', -1, 32)
	res := string(buf.Bytes())
	pool.Put(buf)

	return res
}

// formatDuration converts duration into fractional milliseconds
// with no trailing zeros.
func formatDuration(d time.Duration) string {
	buf := pool.Get()
	buf.AppendUint(uint64(d / time.Millisecond))

	p := uint64(d % time.Millisecond)
	if p > 0 {
		om := 0
		m := uint64(100000)
		for p < m {
			om++
			m /= 10
		}

		for {
			if p%10 == 0 {
				p /= 10
				continue
			}
			break
		}

		buf.WriteByte('.')

		for om > 0 {
			buf.WriteByte('0')
			om--
		}

		buf.AppendUint(p)
	}

	buf.WriteString("ms")
	res := string(buf.Bytes())
	pool.Put(buf)

	return res
}
