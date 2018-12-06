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
			"data: foo\n\n",
		},
		{
			Message{
				event: "error",
				data:  "foo",
			},
			"event: error\ndata: foo\n\n",
		},
		{
			Message{
				id:   "1",
				data: "foo",
			},
			"id: 1\ndata: foo\n\n",
		},
		{
			Message{
				id:    "1",
				retry: 2,
				event: "error",
				data:  "foo",
			},
			"id: 1\nevent: error\nretry: 2\ndata: foo\n\n",
		},
	}

	for _, c := range cases {
		assert.Equal(t, c.msg.String(), c.expected)
	}
}
