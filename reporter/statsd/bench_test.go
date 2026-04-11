package statsd

import (
	"testing"

	"github.com/cactus/go-statsd-client/v5/statsd"
	"github.com/cactus/go-statsd-client/v5/statsd/statsdtest"
)

var benchTags = [][2]string{
	{"env", "production"},
	{"region", "us-east-1"},
	{"service", "api"},
}

func BenchmarkCounter(b *testing.B) {
	sender := statsdtest.NewRecordingSender()
	client, err := statsd.NewClientWithSender(sender, "test", statsd.InfixComma)
	if err != nil {
		b.Fatal(err)
	}
	s := &Statsd{client: client}
	if es, ok := client.(statsd.ExtendedStatSender); ok {
		s.es = es
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Counter("test", 1, benchTags)
		}
	})
}

func BenchmarkGauge(b *testing.B) {
	sender := statsdtest.NewRecordingSender()
	client, err := statsd.NewClientWithSender(sender, "test", statsd.InfixComma)
	if err != nil {
		b.Fatal(err)
	}
	s := &Statsd{client: client}
	if es, ok := client.(statsd.ExtendedStatSender); ok {
		s.es = es
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Gauge("test", 1.5, benchTags)
		}
	})
}
