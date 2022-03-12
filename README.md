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

## Usage

```go
reporter := statsd.New(statsdAddr, "")
stats := statter.New(reporter, 10*time.Second).With("my-prefix")

stats.Counter("my-counter", tags.Str("tag", "value")).Inc(1)
```
