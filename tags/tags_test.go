package tags_test

import (
	"testing"

	"github.com/hamba/statter/v2"
	"github.com/hamba/statter/v2/tags"
	"github.com/stretchr/testify/assert"
)

func TestStr(t *testing.T) {
	tag := tags.Str("key", "val")

	assert.Equal(t, statter.Tag{"key", "val"}, tag)
}

func TestInt(t *testing.T) {
	tag := tags.Int("key", 2)

	assert.Equal(t, statter.Tag{"key", "2"}, tag)
}

func TestInt64(t *testing.T) {
	tag := tags.Int64("key", 2)

	assert.Equal(t, statter.Tag{"key", "2"}, tag)
}

func TestStatusCode(t *testing.T) {
	tag := tags.StatusCode("key", 204)

	assert.Equal(t, statter.Tag{"key", "2xx"}, tag)
}
