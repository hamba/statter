// Package statsd implements a statsd client.
package statsd

import (
	"math"
	"sync"
	"time"

	"github.com/cactus/go-statsd-client/v5/statsd"
)

type config struct {
	flushInterval time.Duration
	flushBytes    int
}

func defaultConfig() config {
	return config{
		flushInterval: 300 * time.Millisecond,
		flushBytes:    1432,
	}
}

// Option represents a statsd option function.
type Option func(*config)

// WithFlushInterval sets the maximum flushInterval for packet sending.
// Defaults to 300ms.
func WithFlushInterval(interval time.Duration) Option {
	return func(c *config) {
		c.flushInterval = interval
	}
}

// WithFlushBytes sets the maximum UDP packet size in bytes that will be sent.
// Defaults to 1432 bytes.
func WithFlushBytes(bytes int) Option {
	return func(c *config) {
		c.flushBytes = bytes
	}
}

// Statsd is a statsd client.
type Statsd struct {
	cfg    config
	client statsd.Statter
	es     statsd.ExtendedStatSender
}

// New returns a statsd reporter.
func New(addr, prefix string, opts ...Option) (*Statsd, error) {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}

	clientCfg := &statsd.ClientConfig{
		Address:       addr,
		Prefix:        prefix,
		UseBuffered:   true,
		FlushInterval: cfg.flushInterval,
		FlushBytes:    cfg.flushBytes,
		TagFormat:     statsd.InfixComma,
	}
	c, err := statsd.NewClientWithConfig(clientCfg)
	if err != nil {
		return nil, err
	}

	s := &Statsd{
		cfg:    cfg,
		client: c,
	}
	if es, ok := c.(statsd.ExtendedStatSender); ok {
		s.es = es
	}
	return s, nil
}

// Counter reports a counter value.
func (s *Statsd) Counter(name string, v int64, tags [][2]string) {
	if len(tags) == 0 {
		_ = s.client.Inc(name, v, 1.0)
		return
	}
	withTags(tags, func(t []statsd.Tag) {
		_ = s.client.Inc(name, v, 1.0, t...)
	})
}

// Gauge reports a gauge value.
//
// If the underlying statsd client supports float gauges (ExtendedStatSender),
// the full float64 value is sent. Otherwise the value is rounded to the
// nearest integer rather than silently truncated.
func (s *Statsd) Gauge(name string, v float64, tags [][2]string) {
	if len(tags) == 0 {
		if s.es != nil {
			_ = s.es.GaugeFloat(name, v, 1.0)
		} else {
			_ = s.client.Gauge(name, int64(math.Round(v)), 1.0)
		}
		return
	}
	withTags(tags, func(t []statsd.Tag) {
		if s.es != nil {
			_ = s.es.GaugeFloat(name, v, 1.0, t...)
		} else {
			_ = s.client.Gauge(name, int64(math.Round(v)), 1.0, t...)
		}
	})
}

// Close closes the client and flushes buffered stats, if applicable.
func (s *Statsd) Close() error {
	return s.client.Close()
}

var tagPool = sync.Pool{
	New: func() any {
		s := make([]statsd.Tag, 0, 8)
		return &s
	},
}

func withTags(tags [][2]string, fn func(t []statsd.Tag)) {
	sp := tagPool.Get().(*[]statsd.Tag)
	t := fillTags(*sp, tags)
	*sp = t
	fn(t)
	tagPool.Put(sp)
}

func fillTags(dst []statsd.Tag, src [][2]string) []statsd.Tag {
	dst = dst[:0]
	for i := range src {
		dst = append(dst, statsd.Tag{src[i][0], src[i][1]})
	}
	return dst
}
