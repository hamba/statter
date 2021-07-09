// Package statsd implements an statsd client.
package statsd

import (
	"time"

	"github.com/cactus/go-statsd-client/v5/statsd"
)

// Option represents statsd option function.
type Option func(*Statsd)

// WithFlushInterval sets the maximum flushInterval for packet sending.
// Defaults to 300ms.
func WithFlushInterval(interval time.Duration) Option {
	return func(s *Statsd) {
		s.flushInterval = interval
	}
}

// WithFlushBytes sets the maximum udp packet size that will be sent.
// Defaults to 1432 flushBytes.
func WithFlushBytes(bytes int) Option {
	return func(s *Statsd) {
		s.flushBytes = bytes
	}
}

// Statsd is a statsd client.
type Statsd struct {
	client statsd.Statter

	flushInterval time.Duration
	flushBytes    int
}

// New returns a statsd reporter.
func New(addr, prefix string, opts ...Option) (*Statsd, error) {
	s := &Statsd{}

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

// Counter reports a counter value.
func (s *Statsd) Counter(name string, v int64, tags [][2]string) {
	_ = s.client.Inc(name, v, 1.0, toTags(tags)...)
}

// Gauge reports a gauge value.
func (s *Statsd) Gauge(name string, v float64, tags [][2]string) {
	_ = s.client.Gauge(name, int64(v), 1.0, toTags(tags)...)
}

// Close closes the client and flushes buffered stats, if applicable.
func (s *Statsd) Close() error {
	return s.client.Close()
}

func toTags(t [][2]string) []statsd.Tag {
	res := make([]statsd.Tag, len(t))
	for i := 0; i < len(t); i++ {
		res[i] = statsd.Tag{t[i][0], t[i][1]}
	}
	return res
}
