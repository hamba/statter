package prometheus_test

import (
	"time"

	"github.com/hamba/statter/v2"
	"github.com/hamba/statter/v2/reporter/prometheus"
)

func ExampleNew() {
	buckets := []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}
	reporter := prometheus.New("my-namespace", prometheus.WithBuckets(buckets))

	statter.New(reporter, 10*time.Second)
}

func ExampleRegisterCounter() {
	reporter := prometheus.New("my-namespace")
	stats := statter.New(reporter, 10*time.Second).With("my-prefix")

	prometheus.RegisterCounter(stats, "my-counter", []string{"tag"}, "my awesome counter")
}

func ExampleRegisterGauge() {
	reporter := prometheus.New("my-namespace")
	stats := statter.New(reporter, 10*time.Second).With("my-prefix")

	prometheus.RegisterGauge(stats, "my-gauge", []string{"tag"}, "my awesome gauge")
}

func ExampleRegisterHistogram() {
	reporter := prometheus.New("my-namespace")
	stats := statter.New(reporter, 10*time.Second).With("my-prefix")

	buckets := []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}
	prometheus.RegisterHistogram(stats, "my-gauge", []string{"tag"}, buckets, "my awesome histogram")
}
