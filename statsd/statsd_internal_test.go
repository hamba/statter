package statsd

import (
	"testing"
	"time"

	"github.com/cactus/go-statsd-client/v5/statsd"
	"github.com/cactus/go-statsd-client/v5/statsd/statsdtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	client, err := statsd.NewClientWithSender(sender, "test", statsd.InfixComma)
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
	client, err := statsd.NewClientWithSender(sender, "test", statsd.InfixComma)
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
	client, err := statsd.NewClientWithSender(sender, "test", statsd.InfixComma)
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
	client, err := statsd.NewClientWithSender(sender, "test", statsd.InfixComma)
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
	client, err := statsd.NewClientWithSender(sender, "test", statsd.InfixComma)
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
	client, err := statsd.NewClientWithSender(sender, "test", statsd.InfixComma)
	assert.NoError(t, err)

	s := &BufferedStatsd{
		client: client,
	}

	s.Timing("test", time.Second, 1.0, "test", "test")

	sent := sender.GetSent()
	require.Len(t, sent, 1)
	assert.Equal(t, "test.test,test=test", sent[0].Stat)
	assert.Equal(t, "1000", sent[0].Value)
}
