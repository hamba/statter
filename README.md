![Logo](http://svg.wiersma.co.za/hamba/project?title=statter&tag=Go%20stats%20clients)

[![Go Report Card](https://goreportcard.com/badge/github.com/hamba/statter)](https://goreportcard.com/report/github.com/hamba/statter)
[![Build Status](https://travis-ci.com/hamba/statter.svg?branch=master)](https://travis-ci.com/hamba/statter)
[![Coverage Status](https://coveralls.io/repos/github/hamba/statter/badge.svg?branch=master)](https://coveralls.io/github/hamba/statter?branch=master)
[![GoDoc](https://godoc.org/github.com/hamba/statter?status.svg)](https://godoc.org/github.com/hamba/statter)
[![GitHub release](https://img.shields.io/github/release/hamba/statter.svg)](https://github.com/hamba/statter/releases)
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/hamba/statter/master/LICENSE)

Go stats clients

## Overview

Install with:

```shell
go get github.com/hamba/statter
```

#### Supported stats clients
* **L2met** Writes l2met to a `Logger` interface
* **Statsd** Writes statsd to `UDP`
* **Prometheus** Exposes stats via `HTTP`
