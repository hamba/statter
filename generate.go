package statter


//go:generate syncmap -pkg prometheus -name counterMap -o reporter/prometheus/maps_counter.gen.go map[string]*prometheus.CounterVec
//go:generate syncmap -pkg prometheus -name gaugeMap -o reporter/prometheus/maps_gauge.gen.go map[string]*prometheus.GaugeVec
//go:generate syncmap -pkg prometheus -name histogramMap -o reporter/prometheus/maps_histogram.gen.go map[string]*prometheus.HistogramVec
