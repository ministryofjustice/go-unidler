package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessageString(t *testing.T) {
	cases := []struct {
		msg      Message
		expected string
	}{
		{
			Message{
				data: "foo",
			},
			"id: \nretry: 0\nevent: \ndata: foo\n\n",
		},
		{
			Message{
				event: "error",
				data:  "foo",
			},
			"id: \nretry: 0\nevent: error\ndata: foo\n\n",
		},
		{
			Message{
				id:   "1",
				data: "foo",
			},
			"id: 1\nretry: 0\nevent: \ndata: foo\n\n",
		},
		{
			Message{
				id:    "1",
				retry: 2,
				event: "error",
				data:  "foo",
			},
			"id: 1\nretry: 2\nevent: error\ndata: foo\n\n",
		},
	}

	for _, c := range cases {
		assert.Equal(t, c.msg.String(), c.expected)
	}
}
