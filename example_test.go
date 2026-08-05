package statter_test

import (
	"time"

	"github.com/hamba/statter/v2"
	"github.com/hamba/statter/v2/tags"
)

func ExampleCounter_Inc() {
	stat := statter.New(statter.DiscardReporter, time.Second)

	stat.Counter("my_counter", tags.Str("tag", "value")).Inc(1)
}

func ExampleGauge_Set() {
	stat := statter.New(statter.DiscardReporter, time.Second)

	stat.Gauge("my_gauge", tags.Int("int", 1)).Set(1.23)
}

func ExampleHistogram_Observe() {
	stat := statter.New(statter.DiscardReporter, time.Second)

	stat.Histogram("my_histo", tags.Str("label", "blah")).Observe(2.34)
}

func ExampleStatter_Scope() {
	stat := statter.New(statter.DiscardReporter, time.Second)

	// The gauge is unique for the given revision. When the revision changes,
	// the gauge reported for the previous revision is removed.
	stat.Scope("build-info", tags.Str("revision", "abc123")).Gauge("info").Set(1)
}

func ExampleScope_Delete() {
	stat := statter.New(statter.DiscardReporter, time.Second)

	scope := stat.Scope("tenant:1", tags.Str("tenant", "1"))
	scope.Counter("requests").Inc(1)
	scope.Timing("latency").Observe(time.Second)

	// Both the counter and the timing are removed.
	scope.Delete()
}
