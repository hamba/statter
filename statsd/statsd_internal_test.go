package statsd

import (
	"testing"
	"time"

	"github.com/cactus/go-statsd-client/statsd"
	"github.com/cactus/go-statsd-client/statsd/statsdtest"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	s, err := New("127.0.0.1:1234", "test")
	assert.NoError(t, err)
	defer s.Close()

	assert.IsType(t, &Statsd{}, s)

	_, err = New("127.0", "test")
	assert.Error(t, err)
}

func TestStatsd_Inc(t *testing.T) {
	sender := statsdtest.NewRecordingSender()
	client, err := statsd.NewClientWithSender(sender, "test")
	assert.NoError(t, err)

	s := &Statsd{
		client: client,
	}

	s.Inc("test", 2, 1.0, "test", "test")

	sent := sender.GetSent()
	assert.Len(t, sent, 1)
	assert.Equal(t, "test.test,test=test", sent[0].Stat)
	assert.Equal(t, "2", sent[0].Value)
}

func TestStatsd_Gauge(t *testing.T) {
	sender := statsdtest.NewRecordingSender()
	client, err := statsd.NewClientWithSender(sender, "test")
	assert.NoError(t, err)

	s := &Statsd{
		client: client,
	}

	s.Gauge("test", 2.0, 1.0, "test", "test")

	sent := sender.GetSent()
	assert.Len(t, sent, 1)
	assert.Equal(t, "test.test,test=test", sent[0].Stat)
	assert.Equal(t, "2", sent[0].Value)
}

func TestStatsd_Timing(t *testing.T) {
	sender := statsdtest.NewRecordingSender()
	client, err := statsd.NewClientWithSender(sender, "test")
	assert.NoError(t, err)

	s := &Statsd{
		client: client,
	}

	s.Timing("test", time.Second, 1.0, "test", "test")

	sent := sender.GetSent()
	assert.Len(t, sent, 1)
	assert.Equal(t, "test.test,test=test", sent[0].Stat)
	assert.Equal(t, "1000", sent[0].Value)
}

func TestNewBuffered(t *testing.T) {
	s, err := NewBuffered("127.0.0.1:1234", "test", WithFlushInterval(time.Second), WithFlushBytes(1))
	assert.NoError(t, err)
	defer s.Close()

	assert.IsType(t, &BufferedStatsd{}, s)
	assert.Equal(t, time.Second, s.flushInterval)
	assert.Equal(t, 1, s.flushBytes)

	_, err = NewBuffered("127.0", "test")
	assert.Error(t, err)
}

func TestBuffered_Inc(t *testing.T) {
	sender := statsdtest.NewRecordingSender()
	client, err := statsd.NewClientWithSender(sender, "test")
	assert.NoError(t, err)

	s := &BufferedStatsd{
		client: client,
	}

	s.Inc("test", 2, 1.0, "test", "test")

	sent := sender.GetSent()
	assert.Len(t, sent, 1)
	assert.Equal(t, "test.test,test=test", sent[0].Stat)
	assert.Equal(t, "2", sent[0].Value)
}

func TestBuffered_Gauge(t *testing.T) {
	sender := statsdtest.NewRecordingSender()
	client, err := statsd.NewClientWithSender(sender, "test")
	assert.NoError(t, err)

	s := &BufferedStatsd{
		client: client,
	}

	s.Gauge("test", 2.0, 1.0, "test", "test")

	sent := sender.GetSent()
	assert.Len(t, sent, 1)
	assert.Equal(t, "test.test,test=test", sent[0].Stat)
	assert.Equal(t, "2", sent[0].Value)
}

func TestBuffered_Timing(t *testing.T) {
	sender := statsdtest.NewRecordingSender()
	client, err := statsd.NewClientWithSender(sender, "test")
	assert.NoError(t, err)

	s := &BufferedStatsd{
		client: client,
	}

	s.Timing("test", time.Second, 1.0, "test", "test")

	sent := sender.GetSent()
	assert.Len(t, sent, 1)
	assert.Equal(t, "test.test,test=test", sent[0].Stat)
	assert.Equal(t, "1000", sent[0].Value)
}

func TestFormatTags(t *testing.T) {
	tags := []interface{}{
		"bool", true,
		"float32", float32(1.23),
		"float64", float64(1.23),
		"int", 1,
		"int8", int8(2),
		"int16", int16(2),
		"int32", int32(2),
		"int64", int64(2),
		"uint", uint(1),
		"uint8", uint8(2),
		"uint16", uint16(2),
		"uint32", uint32(2),
		"uint64", uint64(2),
		"test", "test",
		"foo", "bar",
		"test", "baz",
		"struct", struct {
			Name string
		}{Name: "blah"},
	}

	assert.Equal(t, "", formatTags(nil))
	assert.Equal(t, "", formatTags([]interface{}{}))

	got := formatTags(tags)

	expect := ",bool=true,float32=1.2300000190734863,float64=1.23,int=1,int8=2,int16=2,int32=2,int64=2,uint=1,uint8=2,uint16=2,uint32=2,uint64=2,test=baz,foo=bar,struct={Name:blah}"
	assert.Equal(t, expect, got)
}

func TestFormatTags_Uneven(t *testing.T) {
	tags := []interface{}{
		"test", "test",
		"foo",
	}

	s := formatTags(tags)

	assert.Equal(t, ",test=test,foo=nil,STATTER_ERROR=Normalised odd number of tags by adding nil", s)
}

func BenchmarkFormatTags(b *testing.B) {
	tags := []interface{}{
		"string", "test",
		"float", 1.2,
		"int", 1,
		"bool", true,
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		formatTags(tags)
	}
}
