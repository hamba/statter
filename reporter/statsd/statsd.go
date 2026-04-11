// Package statsd implements an statsd client.
package statsd

import (
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

// Option represents statsd option function.
type Option func(*config)

// WithFlushInterval sets the maximum flushInterval for packet sending.
// Defaults to 300ms.
func WithFlushInterval(interval time.Duration) Option {
	return func(c *config) {
		c.flushInterval = interval
	}
}

// WithFlushBytes sets the maximum udp packet size that will be sent.
// Defaults to 1432 flushBytes.
func WithFlushBytes(bytes int) Option {
	return func(c *config) {
		c.flushBytes = bytes
	}
}

// Statsd is a statsd client.
type Statsd struct {
	cfg    config
	client statsd.Statter
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

	return &Statsd{
		cfg:    cfg,
		client: c,
	}, nil
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
func (s *Statsd) Gauge(name string, v float64, tags [][2]string) {
	if len(tags) == 0 {
		_ = s.client.Gauge(name, int64(v), 1.0)
		return
	}
	withTags(tags, func(t []statsd.Tag) {
		_ = s.client.Gauge(name, int64(v), 1.0, t...)
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
