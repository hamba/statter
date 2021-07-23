package statter

import (
	"sync"
	"unsafe"
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

func newKey(name string, tags []Tag) *key {
	k := keyPool.Get().(*key)
	k.b = k.b[:0]

	k.b = append(k.b, name...)

	// Short path for no tags.
	if len(tags) == 0 {
		return k
	}

	// The tags must be sorted to create a consistent key.
	sortTags(tags)

	for _, tag := range tags {
		k.b = append(k.b, keySep)
		k.b = append(k.b, tag[0]...)
		k.b = append(k.b, keyTagSep)
		k.b = append(k.b, tag[1]...)
	}

	return k
}

// String returns the key as a string.
//
// The returned string should only be used while
// holding a reference to the key, nor should it be
// stored.
func (k *key) String() string {
	return *(*string)(unsafe.Pointer(&k.b))
}

// SafeString returns the key as a string that
// is safe to use after releasing the key.
func (k *key) SafeString() string {
	return string(k.b)
}

func putKey(k *key) {
	keyPool.Put(k)
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
