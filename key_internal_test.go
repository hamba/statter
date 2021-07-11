package statter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKey(t *testing.T) {
	tests := []struct {
		name string

		keyName string
		keyTags []Tag

		want string
	}{
		{
			name:    "key with tags",
			keyName: "some.key",
			keyTags: []Tag{{"first", "tag1"}, {"second", "tag2"}},
			want:    "some.key:first=tag1:second=tag2",
		},
		{
			name:    "key with tags out of order",
			keyName: "some.key",
			keyTags: []Tag{{"second", "tag2"}, {"first", "tag1"}},
			want:    "some.key:first=tag1:second=tag2",
		},
		{
			name:    "key without tags",
			keyName: "some.key",
			keyTags: nil,
			want:    "some.key",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			k := newKey(test.keyName, test.keyTags)
			defer putKey(k)

			assert.Equal(t, test.want, k.String())
		})
	}
}

func BenchmarkKey(b *testing.B) {
	tags := []Tag{{"second", "tag2"}, {"first", "tag1"}, {"third", "tag3"}, {"forth", "tag4"}}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			k := newKey("test", tags)

			putKey(k)
		}
	})
}
