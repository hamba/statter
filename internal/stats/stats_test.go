package stats_test

import (
	"sync"
	"testing"

	"github.com/hamba/statter/v2/internal/stats"
	"github.com/stretchr/testify/assert"
)

func TestPool(t *testing.T) {
	p := stats.NewPool(1000)

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for range 100 {
				s := p.Get()

				assert.Zero(t, s.Count())

				s.Add(12.34)

				assert.Equal(t, int64(1), s.Count())
				assert.Equal(t, 12.34, s.Sum())

				p.Put(s)
			}
		}()
	}

	wg.Wait()
}

func TestSample(t *testing.T) {
	s := stats.NewSample(1000)

	for i := range 1000 * 10 {
		s.Add(float64(i))
	}

	assert.Equal(t, int64(10000), s.Count())
	assert.Equal(t, float64(49995000), s.Sum())
	assert.Equal(t, 4999.5, s.Mean())
	assert.Equal(t, float64(9999), s.Max())
	assert.Equal(t, float64(0), s.Min())
	assert.Equal(t, 2886.751331514372, s.StdDev())
	assert.Equal(t, 8333333.25, s.Variance())
}

func TestSample_Single(t *testing.T) {
	s := stats.NewSample(1000)

	s.Add(12.34)

	assert.Equal(t, int64(1), s.Count())
	assert.Equal(t, 12.34, s.Sum())
	assert.Equal(t, 12.34, s.Mean())
	assert.Equal(t, 12.34, s.Max())
	assert.Equal(t, 12.34, s.Min())
	ps := []float64{12.34, 12.34, 12.34, 12.34, 12.34, 12.34}
	assert.Equal(t, ps, s.Percentiles([]float64{-1, 0, 50, 90, 99.5, 100}))
}

func TestSample_Underflow(t *testing.T) {
	s := stats.NewSample(1000)
	values := []float64{10, 20, 10, 30, 20, 11, 12, 32, 45, 9, 5, 5, 5, 10, 23, 8}

	for _, v := range values {
		s.Add(v)
	}

	assert.Equal(t, int64(16), s.Count())
	assert.Equal(t, float64(255), s.Sum())
	assert.Equal(t, 15.9375, s.Mean())
	assert.Equal(t, float64(45), s.Max())
	assert.Equal(t, float64(5), s.Min())
	ps := []float64{5, 5, 11, 32, 45, 45}
	assert.Equal(t, ps, s.Percentiles([]float64{-1, 0, 50, 90, 99.5, 100}))
}

func BenchmarkSample(b *testing.B) {
	s := stats.NewSample(1000)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		s.Add(12.34)
	}
}
