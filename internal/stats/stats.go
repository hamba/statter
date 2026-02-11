// Package stats implements performant statistics implementations.
package stats

import (
	"math"
	"sort"
	"sync"

	"github.com/valyala/fastrand"
)

// Pool is a pool of samples.
type Pool struct {
	p *sync.Pool
}

// NewPool returns a pool.
func NewPool(percLimit int) *Pool {
	return &Pool{p: &sync.Pool{
		New: func() any {
			return NewSample(percLimit)
		},
	}}
}

// Get returns a sample from the pool, creating one if necessary.
func (p *Pool) Get() *Sample {
	s := p.p.Get().(*Sample)
	s.Reset()
	return s
}

// Put adds a sample to the pool.
func (p *Pool) Put(s *Sample) {
	p.p.Put(s)
}

// Sample calculates incremental statistics such as
// mean, variance, standard deviation and estimates
// percentiles.
//
// The incremental stats are based on the algorithm
// described here:
// https://en.wikipedia.org/wiki/Algorithms_for_calculating_variance .
type Sample struct {
	sum float64
	max float64
	min float64

	k   float64
	n   int64
	ex  float64
	ex2 float64

	perc []float64

	rng fastrand.RNG
}

// NewSample returns a sample with the given percentile
// sample limit.
func NewSample(percLimit int) *Sample {
	return &Sample{
		perc: make([]float64, 0, percLimit),
	}
}

// Add adds a sample value.
func (s *Sample) Add(v float64) {
	if s.n == 0 {
		s.k = v
		s.max = v
		s.min = v
	}

	s.n++
	s.ex += v - s.k
	s.ex2 += (v - s.k) * (v - s.k)

	s.sum += v

	switch {
	case v > s.max:
		s.max = v
	case v < s.min:
		s.min = v
	}

	l, c := len(s.perc), cap(s.perc)
	if l < c {
		s.perc = append(s.perc, v)
	} else if n := int(s.rng.Uint32n(uint32(s.n))); n < l {
		s.perc[n] = v
	}
}

// Reset resets the sample.
func (s *Sample) Reset() {
	s.n = 0
	s.max = 0
	s.min = 0
	s.sum = 0
	s.ex = 0
	s.ex2 = 0
	s.perc = s.perc[:0]
}

// Mean returns the mean of the sample.
func (s *Sample) Mean() float64 {
	return s.k + s.ex/float64(s.n)
}

// Variance returns the variance of the sample.
func (s *Sample) Variance() float64 {
	return (s.ex2 - (s.ex*s.ex)/float64(s.n)) / float64(s.n)
}

// StdDev returns the standard deviation of the sample.
func (s *Sample) StdDev() float64 {
	return math.Sqrt(s.Variance())
}

// Sum returns the sum of the sample.
func (s *Sample) Sum() float64 {
	return s.sum
}

// Max returns the max of the sample.
func (s *Sample) Max() float64 {
	return s.max
}

// Min returns the min of the sample.
func (s *Sample) Min() float64 {
	return s.min
}

// Count returns the number of values in the sample.
func (s *Sample) Count() int64 {
	return s.n
}

// Percentiles returns the estimated percentiles of the sample.
func (s *Sample) Percentiles(ns []float64) []float64 {
	sort.Float64s(s.perc)

	p := make([]float64, len(ns))
	for i, n := range ns {
		p[i] = s.percentile(n)
	}
	return p
}

func (s *Sample) percentile(n float64) float64 {
	i := n / float64(100) * float64(len(s.perc))
	return s.perc[clamp(i, 0, len(s.perc)-1)]
}

func clamp(i float64, minVal, maxVal int) int {
	if i < float64(minVal) {
		return minVal
	}
	if i > float64(maxVal) {
		return maxVal
	}
	return int(i)
}
