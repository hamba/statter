package statter

import (
	"sync"
)

const (
	keySep    = ':'
	keyTagSep = '='
)

var keyPool = sync.Pool{
	New: func() interface{} {
		return &key{b: make([]byte, 0, 256)}
	},
}

type key struct {
	b []byte
}

func newKey(name string, tags []Tag) string {
	k := keyPool.Get().(*key)
	defer keyPool.Put(k)

	k.b = k.b[:0]
	k.b = append(k.b, name...)

	// Short path for no tags.
	if len(tags) == 0 {
		return string(k.b)
	}

	// The tags must be sorted to create a consistent key.
	sortTags(tags)

	for _, tag := range tags {
		k.b = append(k.b, keySep)
		k.b = append(k.b, tag[0]...)
		k.b = append(k.b, keyTagSep)
		k.b = append(k.b, tag[1]...)
	}

	return string(k.b)
}

func sortTags(tags []Tag) {
	var sorted bool
	for !sorted {
		sorted = true
		lmo := len(tags) - 1
		for i := 0; i < lmo; i++ {
			if tags[i+1][0] < tags[i][0] {
				tags[i+1], tags[i] = tags[i], tags[i+1]
				sorted = false
			}
		}
	}
}
