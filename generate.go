package statter

//go:generate syncmap -pkg statter -name counterMap -o maps_counter.gen.go map[string]*Counter
//go:generate syncmap -pkg statter -name gaugeMap -o maps_gauge.gen.go map[string]*Gauge
//go:generate syncmap -pkg statter -name histogramMap -o maps_histogram.gen.go map[string]*Histogram
//go:generate syncmap -pkg statter -name timingMap -o maps_timing.gen.go map[string]*Timing

//go:generate syncmap -pkg prometheus -name counterMap -o reporter/prometheus/maps_counter.gen.go map[string]*prometheus.CounterVec
//go:generate syncmap -pkg prometheus -name gaugeMap -o reporter/prometheus/maps_gauge.gen.go map[string]*prometheus.GaugeVec
//go:generate syncmap -pkg prometheus -name histogramMap -o reporter/prometheus/maps_histogram.gen.go map[string]*prometheus.HistogramVec
//go:generate syncmap -pkg prometheus -name bucketMap -o reporter/prometheus/maps_buckets.gen.go map[string][]float64
