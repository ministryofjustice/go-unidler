package main

import "fmt"

// Message represents a Server Sent Event message
type Message struct {
	data  string
	event string
	group string
	id    string
	retry int
}

func (m *Message) String() string {
	return fmt.Sprintf(`id: %s
retry: %d
event: %s
data: %s

`, m.id, m.retry, m.event, m.data)
}

func sendEvent(s StreamingResponseWriter, m *Message) {
	fmt.Fprintf(s, m.String())
	s.Flush()
}

func sendMessage(s StreamingResponseWriter, msg string) {
	sendEvent(s, &Message{
		data: msg,
	})
}

func sendError(s StreamingResponseWriter, err error) {
	sendEvent(s, &Message{
		event: "error",
		data:  err.Error(),
	})
}
