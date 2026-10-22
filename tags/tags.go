// Package tags provides helpers for creating [statter.Tag] values from common
// Go types.
package tags

import (
	"strconv"

	"github.com/hamba/statter/v2"
)

// Str returns a tag whose value is v.
func Str(k, v string) statter.Tag {
	return [2]string{k, v}
}

// Int returns a tag whose value is the base-10 representation of v.
func Int(k string, v int) statter.Tag {
	return [2]string{k, strconv.Itoa(v)}
}

// Int64 returns a tag whose value is the base-10 representation of v.
func Int64(k string, v int64) statter.Tag {
	return [2]string{k, strconv.FormatInt(v, 10)}
}

// StatusCode returns a tag whose value is the status class, such as "2xx".
func StatusCode(k string, v int) statter.Tag {
	code := strconv.Itoa(v)
	return [2]string{k, string(code[0]) + "xx"}
}
