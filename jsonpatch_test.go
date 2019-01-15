package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonPatchEscape(t *testing.T) {
	assert := assert.New(t)
	cases := []struct {
		value    string
		expected string
	}{
		{"foo", "foo"},
		{"foo/bar", "foo~1bar"},
		{"foo/bar~1", "foo~1bar~01"},
		{"foo/bar/quux/baz", "foo~1bar~1quux~1baz"},
		{"/////", "~1~1~1~1~1"},
		{"~~~~~", "~0~0~0~0~0"},
		{"", ""},
	}

	for _, c := range cases {
		assert.Equal(Escape(c.value), c.expected)
	}
}

func TestJsonPatch(t *testing.T) {
	p := NewJSONPatch(
		Replace("/a/b/c", "bar"),
		Remove("/a/b/d"),
	)
	bytes, err := json.Marshal(p)
	assert := assert.New(t)
	assert.Nil(err)
	assert.Equal(
		`[{"op":"replace","path":"/a/b/c","value":"bar"},{"op":"remove","path":"/a/b/d"}]`,
		string(bytes),
	)
}
