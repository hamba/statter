package l2met_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/hamba/logger/v2"
	"github.com/hamba/statter"
	l2met2 "github.com/hamba/statter/reporter/l2met"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(&buf, logger.LogfmtFormat(), logger.Info)
	r := l2met2.New(log, "test")

	assert.Implements(t, (*statter.Reporter)(nil), r)
}

func TestL2met_Counter(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(&buf, logger.LogfmtFormat(), logger.Info)
	r := l2met2.New(log, "test")

	r.Counter("test", 2, [][2]string{{"foo", "bar"}})

	assert.Equal(t, "lvl=info msg= count#test.test=2 foo=bar\n", buf.String())
}

func TestL2met_Gauge(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(&buf, logger.LogfmtFormat(), logger.Info)
	s := l2met2.New(log, "test")

	s.Gauge("test", 2.1, [][2]string{{"foo", "bar"}})

	assert.Equal(t, "lvl=info msg= sample#test.test=2.100 foo=bar\n", buf.String())
}

func TestL2met_Close(t *testing.T) {
	log := logger.New(io.Discard, logger.LogfmtFormat(), logger.Info)
	s := l2met2.New(log, "test")

	err := s.Close()

	assert.NoError(t, err)
}

func BenchmarkL2met_Inc(b *testing.B) {
	log := logger.New(io.Discard, logger.LogfmtFormat(), logger.Info)
	s := l2met2.New(log, "test")

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Counter("test", 2, [][2]string{{"foo", "bar"}})
		}
	})
}
