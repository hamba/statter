package runtime_test

import (
	"testing"
	"time"

	"github.com/hamba/statter"
	"github.com/hamba/statter/runtime"
	"github.com/stretchr/testify/mock"
)

func TestRuntime(t *testing.T) {
	m := &mockComplexReporter{}
	m.On("Gauge", mock.AnythingOfType("string"), mock.AnythingOfType("float64"), mock.AnythingOfType("[][2]string"))
	m.On("Timing", mock.AnythingOfType("string"), mock.AnythingOfType("[][2]string")).Return(func(_ time.Duration) {})
	stats := statter.New(m, time.Millisecond)
	t.Cleanup(func() { _ = stats.Close() })

	runtime.DefaultRuntimeInterval = time.Microsecond

	go runtime.Collect(stats)

	time.Sleep(100 * time.Millisecond)

	m.AssertCalled(t, "Gauge", "runtime.cpu.goroutines", mock.AnythingOfType("float64"), mock.AnythingOfType("[][2]string"))
}

type mockComplexReporter struct {
	mock.Mock
}

func (r *mockComplexReporter) Counter(name string, v int64, tags [][2]string) {
	_ = r.Called(name, v, tags)
}

func (r *mockComplexReporter) Gauge(name string, v float64, tags [][2]string) {
	_ = r.Called(name, v, tags)
}

func (r *mockComplexReporter) Histogram(name string, tags [][2]string) func(v float64) {
	args := r.Called(name, tags)

	ret := args.Get(0)
	if ret == nil {
		return nil
	}
	return ret.(func(v float64))
}

func (r *mockComplexReporter) Timing(name string, tags [][2]string) func(v time.Duration) {
	args := r.Called(name, tags)

	ret := args.Get(0)
	if ret == nil {
		return nil
	}
	return ret.(func(v time.Duration))
}
