package statter_test

import (
	"context"
	"testing"
	"time"

	"github.com/hamba/statter/v2"
	"github.com/hamba/statter/v2/tags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithContext_ReturnsCtxUnchangedWhenNoTags(t *testing.T) {
	ctx := context.Background()

	got := statter.WithContext(ctx)

	assert.Equal(t, ctx, got)
}

func TestWithContext_AttachesTagsToContext(t *testing.T) {
	m := &mockSimpleReporter{}
	m.On("Counter", "test", int64(1), [][2]string{{"env", "prod"}})

	stats := statter.New(m, time.Second)

	ctx := statter.WithContext(context.Background(), tags.Str("env", "prod"))

	stats.FromContext(ctx).Counter("test").Inc(1)

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestWithContext_MergesWithExistingContextTags(t *testing.T) {
	m := &mockSimpleReporter{}
	m.On("Counter", "test", int64(1), [][2]string{{"env", "prod"}, {"region", "us-east"}})

	stats := statter.New(m, time.Second)

	ctx := statter.WithContext(context.Background(), tags.Str("env", "prod"))
	ctx = statter.WithContext(ctx, tags.Str("region", "us-east"))

	stats.FromContext(ctx).Counter("test").Inc(1)

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestWithContext_NewTagOverridesExistingTagWithSameKey(t *testing.T) {
	m := &mockSimpleReporter{}
	m.On("Counter", "test", int64(1), [][2]string{{"env", "staging"}})

	stats := statter.New(m, time.Second)

	ctx := statter.WithContext(context.Background(), tags.Str("env", "prod"))
	ctx = statter.WithContext(ctx, tags.Str("env", "staging"))

	stats.FromContext(ctx).Counter("test").Inc(1)

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_FromContextReturnsStatterWhenNoContextTags(t *testing.T) {
	stats := statter.New(statter.DiscardReporter, time.Second)
	t.Cleanup(func() { _ = stats.Close() })

	got := stats.FromContext(context.Background())

	assert.Same(t, stats, got)
}

func TestStatter_FromContextMergesContextTagsWithStatterBaseTags(t *testing.T) {
	m := &mockSimpleReporter{}
	m.On("Counter", "test", int64(1), [][2]string{{"base", "val"}, {"env", "prod"}})

	stats := statter.New(m, time.Second, statter.WithTags(tags.Str("base", "val")))

	ctx := statter.WithContext(context.Background(), tags.Str("env", "prod"))

	stats.FromContext(ctx).Counter("test").Inc(1)

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestStatter_FromContextContextTagOverridesStatterBaseTag(t *testing.T) {
	m := &mockSimpleReporter{}
	m.On("Counter", "test", int64(1), [][2]string{{"env", "staging"}})

	stats := statter.New(m, time.Second, statter.WithTags(tags.Str("env", "prod")))

	ctx := statter.WithContext(context.Background(), tags.Str("env", "staging"))

	stats.FromContext(ctx).Counter("test").Inc(1)

	err := stats.Close()
	require.NoError(t, err)

	m.AssertExpectations(t)
}
