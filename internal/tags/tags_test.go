package tags_test

import (
	"testing"

	"github.com/hamba/statter/internal/tags"
	"github.com/stretchr/testify/assert"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name string
		tags []interface{}
		want []interface{}
	}{
		{
			name: "Valid",
			tags: []interface{}{"test1", "foo"},
			want: []interface{}{"test1", "foo"},
		},
		{
			name: "Unpaired",
			tags: []interface{}{"test1"},
			want: []interface{}{"test1", nil, "STATTER_ERROR", "Normalised odd number of tags by adding nil"},
		},
	}

	for _, test := range tests {
		got := tags.Normalize(test.tags)

		assert.Equal(t, test.want, got)
	}
}

func TestDeduplicate(t *testing.T) {
	tests := []struct {
		name string
		tags []interface{}
		want []interface{}
	}{
		{
			name: "Duplicates",
			tags: []interface{}{"test1", "foo", "test1", "bar"},
			want: []interface{}{"test1", "bar"},
		},
		{
			name: "No Duplicates",
			tags: []interface{}{"test1", "foo", "test2", "bar"},
			want: []interface{}{"test1", "foo", "test2", "bar"},
		},
		{
			name: "Duplicate Ordering",
			tags: []interface{}{"test1", "foo", "test2", "bar", "test1", "baz"},
			want: []interface{}{"test1", "baz", "test2", "bar"},
		},
	}

	for _, test := range tests {
		got := tags.Deduplicate(test.tags)

		assert.Equal(t, test.want, got)
	}
}

func BenchmarkTaggedStats_DeduplicateTags(b *testing.B) {
	t := []interface{}{
		"test1", "foo",
		"test2", "bar",
		"test1", "baz",
		"test3", "test",
		"test4", "test",
		"test5", "test",
	}

	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		tags.Deduplicate(t)
	}
}
