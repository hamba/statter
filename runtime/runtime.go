package runtime

import (
	"runtime"
	"time"

	"github.com/hamba/statter"
)

// DefaultRuntimeInterval is the default runtime collection interval.
var DefaultRuntimeInterval = 10 * time.Second

// Collect collects runtime metrics periodically sending them to s.
func Collect(s *statter.Statter) {
	CollectEvery(s, DefaultRuntimeInterval)
}

// CollectEvery collects runtime metrics at interval d sending them to s.
func CollectEvery(s *statter.Statter, d time.Duration) {
	c := time.Tick(d)
	for range c {
		r := newRuntimeStats()
		r.send(s)
	}
}

type runtimeStats struct {
	*runtime.MemStats

	goroutines int
}

func newRuntimeStats() *runtimeStats {
	r := &runtimeStats{MemStats: &runtime.MemStats{}}
	runtime.ReadMemStats(r.MemStats)
	r.goroutines = runtime.NumGoroutine()

	return r
}

func (r *runtimeStats) send(s *statter.Statter) {
	// CPU
	s.Gauge("runtime.cpu.goroutines").Set(float64(r.goroutines))

	// Memory
	// General
	s.Gauge("runtime.memory.alloc").Set(float64(r.MemStats.Alloc))
	s.Gauge("runtime.memory.total").Set(float64(r.MemStats.TotalAlloc))
	s.Gauge("runtime.memory.sys").Set(float64(r.MemStats.Sys))
	s.Gauge("runtime.memory.lookups").Set(float64(r.MemStats.Lookups))
	s.Gauge("runtime.memory.mallocs").Set(float64(r.MemStats.Mallocs))
	s.Gauge("runtime.memory.frees").Set(float64(r.MemStats.Frees))

	// Heap
	s.Gauge("runtime.memory.heap.alloc").Set(float64(r.MemStats.HeapAlloc))
	s.Gauge("runtime.memory.heap.sys").Set(float64(r.MemStats.HeapSys))
	s.Gauge("runtime.memory.heap.idle").Set(float64(r.MemStats.HeapIdle))
	s.Gauge("runtime.memory.heap.inuse").Set(float64(r.MemStats.HeapInuse))
	s.Gauge("runtime.memory.heap.objects").Set(float64(r.MemStats.HeapObjects))
	s.Gauge("runtime.memory.heap.released").Set(float64(r.MemStats.HeapReleased))

	// Stack
	s.Gauge("runtime.memory.stack.inuse").Set(float64(r.MemStats.StackInuse))
	s.Gauge("runtime.memory.stack.sys").Set(float64(r.MemStats.StackSys))
	s.Gauge("runtime.memory.stack.mcache_inuse").Set(float64(r.MemStats.MCacheInuse))
	s.Gauge("runtime.memory.stack.mcache_sys").Set(float64(r.MemStats.MCacheSys))
	s.Gauge("runtime.memory.stack.mspan_inuse").Set(float64(r.MemStats.MSpanInuse))
	s.Gauge("runtime.memory.stack.mspan_sys").Set(float64(r.MemStats.MSpanSys))

	// GC
	s.Gauge("runtime.memory.gc.last").Set(float64(r.MemStats.LastGC))
	s.Gauge("runtime.memory.gc.next").Set(float64(r.MemStats.NextGC))
	s.Gauge("runtime.memory.gc.count").Set(float64(r.MemStats.NumGC))
	s.Timing("runtime.memory.gc.pause").Observe(time.Duration(r.MemStats.PauseTotalNs))
}
