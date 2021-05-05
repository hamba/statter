// Package statsd implements an statsd client.
package statsd

import (
	"time"

	"github.com/cactus/go-statsd-client/v5/statsd"
	"github.com/hamba/statter/internal/tags"
)

// Statsd is a statsd client.
type Statsd struct {
	client statsd.Statter
}

// New returns a statsd instance.
func New(addr, prefix string) (*Statsd, error) {
	config := &statsd.ClientConfig{
		Address:     addr,
		Prefix:      prefix,
		UseBuffered: false,
		TagFormat:   statsd.InfixSemicolon,
	}
	c, err := statsd.NewClientWithConfig(config)
	if err != nil {
		return nil, err
	}

	return &Statsd{
		client: c,
	}, nil
}

// Inc increments a count by the value.
func (s *Statsd) Inc(name string, value int64, rate float32, tags ...string) {
	_ = s.client.Inc(name, value, rate, toTags(tags)...)
}

// Gauge measures the value of a metric.
func (s *Statsd) Gauge(name string, value float64, rate float32, tags ...string) {
	_ = s.client.Gauge(name, int64(value), rate, toTags(tags)...)
}

// Timing sends the value of a Duration.
func (s *Statsd) Timing(name string, value time.Duration, rate float32, tags ...string) {
	_ = s.client.TimingDuration(name, value, rate, toTags(tags)...)
}

// Close closes the client and flushes buffered stats, if applicable.
func (s *Statsd) Close() error {
	return s.client.Close()
}

// BufferedStatsdFunc represents an configuration function for BufferedStatsd.
type BufferedStatsdFunc func(*BufferedStatsd)

// WithFlushInterval sets the maximum flushInterval for packet sending.
// Defaults to 300ms.
func WithFlushInterval(interval time.Duration) BufferedStatsdFunc {
	return func(s *BufferedStatsd) {
		s.flushInterval = interval
	}
}

// WithFlushBytes sets the maximum udp packet size that will be sent.
// Defaults to 1432 flushBytes.
func WithFlushBytes(bytes int) BufferedStatsdFunc {
	return func(s *BufferedStatsd) {
		s.flushBytes = bytes
	}
}

// BufferedStatsd represents a buffered statsd client.
type BufferedStatsd struct {
	client statsd.Statter

	flushInterval time.Duration
	flushBytes    int
}

// NewBuffered create a buffered Statsd instance.
func NewBuffered(addr, prefix string, opts ...BufferedStatsdFunc) (*BufferedStatsd, error) {
	s := &BufferedStatsd{}

	for _, o := range opts {
		o(s)
	}

	config := &statsd.ClientConfig{
		Address:       addr,
		Prefix:        prefix,
		UseBuffered:   true,
		FlushInterval: s.flushInterval,
		FlushBytes:    s.flushBytes,
		TagFormat:     statsd.InfixComma,
	}
	c, err := statsd.NewClientWithConfig(config)
	if err != nil {
		return nil, err
	}
	s.client = c

	return s, nil
}

// Inc increments a count by the value.
func (s *BufferedStatsd) Inc(name string, value int64, rate float32, tags ...string) {
	_ = s.client.Inc(name, value, rate, toTags(tags)...)
}

// Gauge measures the value of a metric.
func (s *BufferedStatsd) Gauge(name string, value float64, rate float32, tags ...string) {
	_ = s.client.Gauge(name, int64(value), rate, toTags(tags)...)
}

// Timing sends the value of a Duration.
func (s *BufferedStatsd) Timing(name string, value time.Duration, rate float32, tags ...string) {
	_ = s.client.TimingDuration(name, value, rate, toTags(tags)...)
}

// Close closes the client and flushes buffered stats, if applicable.
func (s *BufferedStatsd) Close() error {
	return s.client.Close()
}

func toTags(t []string) []statsd.Tag {
	t = tags.Normalize(t)

	res := make([]statsd.Tag, 0, len(t)/2)
	for i := 0; i < len(t); i += 2 {
		res = append(res, statsd.Tag{t[i], t[i+1]})
	}
	return res
}
