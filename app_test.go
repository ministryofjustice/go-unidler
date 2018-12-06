package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonPatchEscape(t *testing.T) {
	assert := assert.New(t)
	cases := []struct {
		value    string
		expected string
	}{
		{
			"foo",
			"foo",
		},
		{
			"foo/bar",
			"foo~1bar",
		},
		{
			"foo/bar~1",
			"foo~1bar~01",
		},
		{
			"foo/bar/quux/baz",
			"foo~1bar~1quux~1baz",
		},
		{
			"/////",
			"~1~1~1~1~1",
		},
		{
			"~~~~~",
			"~0~0~0~0~0",
		},
		{
			"",
			"",
		},
	}

	for _, c := range cases {
		assert.Equal(jsonPatchEscape(c.value), c.expected)
	}
}
