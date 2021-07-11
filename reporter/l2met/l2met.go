// Package l2met implements an l2met stats client.
package l2met

import (
	"github.com/hamba/logger/v2"
	"github.com/hamba/logger/v2/ctx"
	"github.com/hamba/statter/v2/internal/bytes"
)

// L2met is a l2met client.
type L2met struct {
	log    *logger.Logger
	prefix string

	pool bytes.Pool
}

// New returns a l2met reporter.
func New(log *logger.Logger, prefix string) *L2met {
	if len(prefix) > 0 {
		prefix += "."
	}

	s := &L2met{
		log:    log,
		prefix: prefix,
		pool:   bytes.NewPool(512),
	}

	return s
}

// Counter reports a counter value.
func (l *L2met) Counter(name string, v int64, tags [][2]string) {
	k := l.key("count", name)

	l.render(ctx.Int64(k, v), tags)
}

// Gauge reports a gauge value.
func (l *L2met) Gauge(name string, v float64, tags [][2]string) {
	k := l.key("sample", name)

	l.render(ctx.Float64(k, v), tags)
}

func (l *L2met) key(measure, name string) string {
	buf := l.pool.Get()
	buf.WriteString(measure)
	buf.WriteByte('#')
	buf.WriteString(l.prefix)
	buf.WriteString(name)
	str := string(buf.Bytes())
	l.pool.Put(buf)

	return str
}

// render outputs the metric to the logger.
func (l *L2met) render(field logger.Field, t [][2]string) {
	fields := make([]logger.Field, len(t)+1)
	fields[0] = field
	for i, tag := range t {
		fields[i+1] = ctx.Str(tag[0], tag[1])
	}

	l.log.Info("", fields...)
}

// Close closes the client and flushes buffered stats, if applicable.
func (l *L2met) Close() error {
	return nil
}
