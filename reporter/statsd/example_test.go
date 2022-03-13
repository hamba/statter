package statsd_test

import (
	"time"

	"github.com/hamba/statter/v2"
	"github.com/hamba/statter/v2/reporter/statsd"
)

func ExampleNew() {
	reporter, err := statsd.New("127.0.0.1:8125", "my-prefix",
		statsd.WithFlushBytes(1432),
		statsd.WithFlushInterval(300*time.Millisecond),
	)
	if err != nil {
		panic(err)
	}

	statter.New(reporter, 10*time.Second)
}
