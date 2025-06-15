package statter_test

import (
	"testing"
	"time"

	"github.com/hamba/statter/v2"
	"github.com/hamba/statter/v2/reporter/prometheus"
	"github.com/hamba/statter/v2/tags"
)

func BenchmarkStatter_Counter(b *testing.B) {
	s := statter.New(discardReporter{}, time.Second)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Counter("test", tags.Str("test", "test")).Inc(1)
		}
	})

	b.StopTimer()
	_ = s.Close()
}

func BenchmarkStatter_Gauge(b *testing.B) {
	s := statter.New(discardReporter{}, time.Second)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Gauge("test", tags.Str("test", "test")).Set(1)
		}
	})

	b.StopTimer()
	_ = s.Close()
}

func BenchmarkStatter_GaugeAdd(b *testing.B) {
	s := statter.New(discardReporter{}, time.Second)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Gauge("test", tags.Str("test", "test")).Add(1)
		}
	})

	b.StopTimer()
	_ = s.Close()
}

func BenchmarkStatter_Histogram(b *testing.B) {
	s := statter.New(discardReporter{}, time.Second)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Histogram("test", tags.Str("test", "test")).Observe(12.34)
		}
	})

	b.StopTimer()
	_ = s.Close()
}

func BenchmarkStatter_Timing(b *testing.B) {
	s := statter.New(discardReporter{}, time.Second)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Timing("test", tags.Str("test", "test")).Observe(12340 * time.Microsecond)
		}
	})

	b.StopTimer()
	_ = s.Close()
}

func BenchmarkStatter_PrometheusHistogram(b *testing.B) {
	s := statter.New(prometheus.New("test"), time.Second)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Histogram("test", tags.Str("test", "test")).Observe(12.34)
		}
	})

	b.StopTimer()
	_ = s.Close()
}

func BenchmarkStatter_PrometheusTiming(b *testing.B) {
	s := statter.New(prometheus.New("test"), time.Second)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Timing("test", tags.Str("test", "test")).Observe(12340 * time.Microsecond)
		}
	})

	b.StopTimer()
	_ = s.Close()
}

type discardReporter struct{}

func (r discardReporter) Counter(string, int64, [][2]string) {}

func (r discardReporter) Gauge(string, float64, [][2]string) {}
