![Logo](http://svg.wiersma.co.za/hamba/project?title=statter&tag=Go%20stats%20clients)

[![Go Report Card](https://goreportcard.com/badge/github.com/hamba/statter)](https://goreportcard.com/report/github.com/hamba/statter)
[![Build Status](https://github.com/hamba/statter/actions/workflows/test.yml/badge.svg)](https://github.com/hamba/statter/actions)
[![Coverage Status](https://coveralls.io/repos/github/hamba/statter/badge.svg?branch=master)](https://coveralls.io/github/hamba/statter?branch=master)
[![Go Reference](https://pkg.go.dev/badge/github.com/hamba/statter/v2.svg)](https://pkg.go.dev/github.com/hamba/statter/v2)
[![GitHub release](https://img.shields.io/github/release/hamba/statter.svg)](https://github.com/hamba/statter/releases)
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/hamba/statter/master/LICENSE)

Go stats clients

## Overview

Install with:

```shell
go get github.com/hamba/statter/v2
```

#### Supported stats clients
* **L2met** Writes l2met to a `Logger` interface
* **Statsd** Writes statsd to `UDP`
* **Prometheus** Exposes stats via `HTTP`
* **VictoriaMetrics** Exposes stats via `HTTP`

**Note:** This project has renamed the default branch from `master` to `main`. You will need to update your local environment.

## Usage

```go
reporter := statsd.New(statsdAddr, "")
stats := statter.New(reporter, 10*time.Second).With("my-prefix")

stats.Counter("my-counter", tags.Str("tag", "value")).Inc(1)
```

### Scopes

A `Scope` is a sub-statter identified by an id, which tracks the metrics created
through it.

Requesting a scope with an existing id but different tags removes the previous
scope and every metric created through it. This keeps a metric unique for a set
of tags, such as a gauge carrying a revision that changes over time:

```go
stats.Scope("build-info", tags.Str("revision", rev)).Gauge("info").Set(1)
```

The tracked metrics can also be removed together as a batch:

```go
scope := stats.Scope("tenant:"+id, tags.Str("tenant", id))
scope.Counter("requests").Inc(1)
scope.Timing("latency").Observe(d)

scope.Delete() // Both metrics are removed.
```

Use `HasScope` to determine if a scope currently exists.

**Note:** Metrics are only removed from the backend if the reporter supports
removal, which the Prometheus and VictoriaMetrics reporters do. Otherwise
deletion only stops local aggregation.
