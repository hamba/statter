package statsd

import (
	"testing"
	"time"

	"github.com/cactus/go-statsd-client/v5/statsd"
	"github.com/cactus/go-statsd-client/v5/statsd/statsdtest"
	"github.com/hamba/statter/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	s, err := New("127.0.0.1:1234", "test", WithFlushInterval(time.Second), WithFlushBytes(1))
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	assert.Implements(t, (*statter.Reporter)(nil), s)
	assert.Equal(t, time.Second, s.cfg.flushInterval)
	assert.Equal(t, 1, s.cfg.flushBytes)

	_, err = New("127.0", "test")
	assert.Error(t, err)
}

func TestNew_Defaults(t *testing.T) {
	s, err := New("127.0.0.1:1234", "test")
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	assert.Implements(t, (*statter.Reporter)(nil), s)
	assert.Equal(t, 300*time.Millisecond, s.cfg.flushInterval)
	assert.Equal(t, 1432, s.cfg.flushBytes)

	_, err = New("127.0", "test")
	assert.Error(t, err)
}

func TestStatsd_Counter(t *testing.T) {
	sender := statsdtest.NewRecordingSender()
	client, err := statsd.NewClientWithSender(sender, "test", statsd.InfixComma)
	require.NoError(t, err)

	s := &Statsd{
		client: client,
	}

	s.Counter("test", 2, [][2]string{{"test", "test"}})

	sent := sender.GetSent()
	assert.Len(t, sent, 1)
	assert.Equal(t, "test.test,test=test", sent[0].Stat)
	assert.Equal(t, "2", sent[0].Value)
}

func TestStatsd_Gauge(t *testing.T) {
	sender := statsdtest.NewRecordingSender()
	client, err := statsd.NewClientWithSender(sender, "test", statsd.InfixComma)
	require.NoError(t, err)

	s := &Statsd{
		client: client,
	}

	s.Gauge("test", 2.0, [][2]string{{"test", "test"}})

	sent := sender.GetSent()
	assert.Len(t, sent, 1)
	assert.Equal(t, "test.test,test=test", sent[0].Stat)
	assert.Equal(t, "2", sent[0].Value)
}
