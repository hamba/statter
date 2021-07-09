package statter_test

import (
	"testing"
	"time"

	"github.com/hamba/statter"
	"github.com/hamba/statter/reporter/prometheus"
	"github.com/hamba/statter/tags"
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

func (r discardReporter) Counter(name string, v int64, tags [][2]string) {}

func (r discardReporter) Gauge(name string, v float64, tags [][2]string) {}
