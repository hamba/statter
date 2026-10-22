// Package statter collects and reports statistics via a pluggable [Reporter].
//
// Stats are collected in memory and flushed to the reporter on a fixed
// interval. Counters report their accumulated delta and reset after each
// flush; gauges retain and report their last-set value. Histograms and timings
// are either delegated directly to the reporter (when it implements
// [HistogramReporter] or [TimingReporter]) or aggregated locally and emitted
// as derived gauges and a count counter.
//
// Metrics are identified by a name and an optional set of key/value [Tag]
// pairs. The [Statter.With] method creates a scoped sub-statter that
// prepends a prefix and merges tags into every metric it records. Sub-statters
// with identical resolved prefix and tags are deduplicated and share the same
// instance.
//
// The [Statter.Scope] method returns a [Scope], a sub-statter identified by a
// caller-supplied ID. Newly registered metrics created through a Scope are
// tracked and can be removed together with [Scope.Delete]. Requesting a Scope
// with an existing ID but different tags or a different parent deletes the
// previous Scope and its metrics. This keeps metrics unique for a set of tags,
// such as a gauge carrying a revision that changes over time.
//
// Backend adapters must satisfy [Reporter]. They may additionally implement
// [HistogramReporter] and [TimingReporter] to receive observations directly,
// or the corresponding Removable* interfaces to remove metrics from the
// backend. Without a removable interface, deletion stops local aggregation
// but cannot remove values already reported to the backend.
package statter
