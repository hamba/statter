package statter_test

import (
	"time"

	"github.com/hamba/statter/v2"
	"github.com/hamba/statter/v2/tags"
)

func ExampleCounter_Inc() {
	stat := statter.New(nil, time.Second)

	stat.Counter("my_counter", tags.Str("tag", "value")).Inc(1)
}

func ExampleGauge_Set() {
	stat := statter.New(nil, time.Second)

	stat.Gauge("my_gauge", tags.Int("int", 1)).Set(1.23)
}

func ExampleHistogram_Observe() {
	stat := statter.New(nil, time.Second)

	stat.Histogram("my_histo", tags.Str("label", "blah")).Observe(2.34)
}
