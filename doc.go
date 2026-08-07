// Package statter collects and reports statistics via a pluggable [Reporter].
//
// Stats are aggregated in memory and flushed to the reporter on a fixed
// interval. Counters accumulate deltas between flushes; gauges hold their
// last-set value. Histograms and timings are either delegated directly to
// the reporter (when it implements [HistogramReporter] / [TimingReporter])
// or aggregated locally and emitted as a set of derived gauges and a counter.
//
// Metrics are identified by a name and an optional set of key/value [Tag]
// pairs. The [Statter.With] method creates a scoped sub-statter that
// prepends a prefix and merges tags into every metric it records. Sub-statters
// with identical resolved prefix and tags are deduplicated and share the same
// instance.
//
// The [Statter.Scope] method returns a [Scope], a sub-statter identified by a
// caller supplied id. A Scope tracks the metrics created through it, so they
// can be removed together with [Scope.Delete]. Requesting a Scope with an
// existing id but different tags deletes the previous Scope and its metrics,
// which keeps metrics unique for a set of tags, such as a gauge carrying a
// revision that changes over time.
//
// The [Reporter] interface is the only contract that backend adapters must
// satisfy. Richer adapters may additionally implement [HistogramReporter],
// [TimingReporter], and the corresponding Removable* interfaces. Metrics are
// only removed from the backend if the reporter implements the relevant
// Removable* interface, otherwise deletion only stops local aggregation.
package statter
