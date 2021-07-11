// Package tags implements statter tags convenience functions.
package tags

import (
	"strconv"

	"github.com/hamba/statter/v2"
)

// Str returns a string tag with the give key and value.
func Str(k, v string) statter.Tag {
	return [2]string{k, v}
}

// Int returns an int tag with the give key and value.
func Int(k string, v int) statter.Tag {
	return [2]string{k, strconv.Itoa(v)}
}

// Int64 returns an int64 tag with the give key and value.
func Int64(k string, v int64) statter.Tag {
	return [2]string{k, strconv.Itoa(int(v))}
}

// StatusCode return a tag with the given key and the status
// code int the form '2xx'.
func StatusCode(k string, v int) statter.Tag {
	code := strconv.Itoa(v)
	return [2]string{k, string(code[0]) + "xx"}
}
