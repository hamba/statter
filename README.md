![Logo](http://svg.wiersma.co.za/hamba/project?title=statter&tag=Go%20stats%20clients)

[![Go Report Card](https://goreportcard.com/badge/github.com/hamba/statter)](https://goreportcard.com/report/github.com/hamba/statter)
[![Build Status](https://github.com/hamba/statter/actions/workflows/test.yml/badge.svg)](https://github.com/hamba/statter/actions)
[![Coverage Status](https://coveralls.io/repos/github/hamba/statter/badge.svg?branch=master)](https://coveralls.io/github/hamba/statter?branch=master)
[![Go Reference](https://pkg.go.dev/badge/github.com/hamba/statter/v2.svg)](https://pkg.go.dev/github.com/hamba/statter/v2)
[![GitHub release](https://img.shields.io/github/release/hamba/statter.svg)](https://github.com/hamba/statter/releases)
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/hamba/statter/master/LICENSE)

`statter` is a Go library for recording counters, gauges, histograms, and
timings through interchangeable reporters.

## Install

```shell
go get github.com/hamba/statter/v2
```

## Quick start

Create a reporter, pass it to `statter.New` with a positive flush interval, and
close the root statter when the application is shutting down. Closing flushes
pending values and closes the reporter if it implements `io.Closer`.

```go
package main

import (
	"time"

	"github.com/hamba/statter/v2"
	"github.com/hamba/statter/v2/reporter/statsd"
	"github.com/hamba/statter/v2/tags"
)

func main() {
	reporter, err := statsd.New("127.0.0.1:8125", "")
	if err != nil {
		panic(err)
	}

	root := statter.New(reporter, 10*time.Second)
	defer root.Close()

	stats := root.With("my-service")
	stats.Counter("requests", tags.Str("route", "/")).Inc(1)
	stats.Gauge("queue-depth").Set(3)
	stats.Histogram("response-size").Observe(512)
	stats.Timing("request-duration").Observe(25 * time.Millisecond)
}
```

Counters report the accumulated increment since the previous flush and reset
after each flush. Gauges retain their last value and are reported on every
flush. Histograms and timings are sent directly when the reporter supports
them; otherwise, statter aggregates observations between flushes and reports
count, sum, mean, standard deviation, minimum, maximum, and configured
percentiles.

## Names and tags

Use `With` to apply a shared name prefix and tags to metrics. Prefixes are joined
using `.` by default, and tags with the same key are overridden by the newer
value. The separator can be changed with `WithSeparator`.

```go
serviceStats := stats.With("worker", tags.Str("region", "west"))
serviceStats.Counter("jobs", tags.Str("queue", "email")).Inc(1)
```

`WithContext` attaches tags to a context, and `FromContext` adds those tags to a
statter. This is useful for carrying request-specific tags through functions
without passing a scoped statter explicitly.

## Scopes

A `Scope` tracks the metrics created through it so that they can be removed as
a group. Scope IDs are unique across the statter tree. Requesting an existing
ID with different tags or a different parent deletes its previous metrics and
replaces the scope. This is useful for metrics that represent a replaceable
set of labels, such as build or revision information:

```go
stats.Scope("build-info", tags.Str("revision", revision)).
Gauge("info").
Set(1)
```

Delete a scope explicitly when its metrics are no longer relevant:

```go
scope := stats.Scope("tenant:"+tenantID, tags.Str("tenant", tenantID))
scope.Counter("requests").Inc(1)
scope.Timing("latency").Observe(25 * time.Millisecond)
scope.Delete()
```

Metrics are removed from the backend only when the reporter supports removal.
Otherwise deletion stops local aggregation, but the reporter may retain the
last values it received.

Only metrics first registered through the Scope are tracked by it. If the same
metric was registered before it was requested through a Scope, deleting the
Scope does not remove that pre-existing metric.

## Reporters

The base `Reporter` interface handles counters and gauges. A reporter may also
implement histogram and timing interfaces to receive those observations
directly, and removable interfaces to remove backend metrics when they are
deleted.

| Reporter                                                                                              | Backend                                                      |
|-------------------------------------------------------------------------------------------------------|--------------------------------------------------------------|
| [`reporter/l2met`](https://pkg.go.dev/github.com/hamba/statter/v2/reporter/l2met)                     | Writes metrics to a `hamba/logger` logger                    |
| [`reporter/statsd`](https://pkg.go.dev/github.com/hamba/statter/v2/reporter/statsd)                   | Sends metrics to StatsD over UDP                             |
| [`reporter/prometheus`](https://pkg.go.dev/github.com/hamba/statter/v2/reporter/prometheus)           | Exposes metrics through an HTTP handler                      |
| [`reporter/victoriametrics`](https://pkg.go.dev/github.com/hamba/statter/v2/reporter/victoriametrics) | Exposes metrics in Prometheus format through an HTTP handler |
