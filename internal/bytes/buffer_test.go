package bytes_test

import (
	"sync"
	"testing"
	"time"

	"github.com/hamba/statter/v2/internal/bytes"
	"github.com/stretchr/testify/assert"
)

func TestPool(t *testing.T) {
	const dummyData = "dummy data"

	p := bytes.NewPool(512)

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for range 100 {
				buf := p.Get()
				assert.Zero(t, buf.Len(), "Expected truncated Buffer")
				assert.NotZero(t, buf.Cap(), "Expected non-zero capacity")

				buf.WriteString(dummyData)

				assert.Len(t, dummyData, buf.Len(), "Expected Buffer to contain dummy data")

				p.Put(buf)
			}
		}()
	}

	wg.Wait()
}

func TestBuffer(t *testing.T) {
	buf := bytes.NewPool(512).Get()

	tests := []struct {
		name string
		fn   func()
		want string
	}{
		{
			name: "WriteByte",
			fn:   func() { buf.WriteByte('v') },
			want: "v",
		},
		{
			name: "WriteString",
			fn:   func() { buf.WriteString("foo") },
			want: "foo",
		},
		{
			name: "Write",
			fn:   func() { buf.Write([]byte("foo")) },
			want: "foo",
		},
		{
			name: "AppendIntPositive",
			fn:   func() { buf.AppendInt(42) }, want: "42",
		},
		{
			name: "AppendIntNegative",
			fn:   func() { buf.AppendInt(-42) }, want: "-42",
		},
		{
			name: "AppendUint",
			fn:   func() { buf.AppendUint(42) }, want: "42",
		},
		{
			name: "AppendBool",
			fn:   func() { buf.AppendBool(true) }, want: "true",
		},
		{
			name: "AppendFloat64",
			fn:   func() { buf.AppendFloat(3.14, 'f', 3, 64) },
			want: "3.140",
		},
		{
			name: "AppendTime",
			fn:   func() { buf.AppendTime(time.Unix(1541573670, 0).UTC(), time.RFC3339) },
			want: "2018-11-07T06:54:30Z",
		},
	}

	assert.Equal(t, 512, buf.Cap())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			tt.fn()

			assert.Equal(t, tt.want, string(buf.Bytes()))
			assert.Equal(t, len(tt.want), buf.Len())
		})
	}
}
