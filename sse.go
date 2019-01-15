package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
)

type (
	// Message represents a Server Sent Event message
	Message struct {
		event string
		data  string
		id    string
		retry int
	}

	// SSESender provides a method to send a SSE event to a specified channel
	SSESender interface {
		SendSSE(*Message)
	}

	// SSEBroker represents the server with a list of channels
	SSEBroker struct {
		lastMessage *Message
		mutex       *sync.Mutex
		send        chan string
	}
)

func (m *Message) String() string {
	return fmt.Sprintf(`id: %s
retry: %d
event: %s
data: %s

`, m.id, m.retry, m.event, m.data)
}

// NewSSEBroker constructs a new SSE server and starts it running
func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		mutex: &sync.Mutex{},
		send:  make(chan string),
	}
}

// ServeHTTP receives HTTP requests from browsers and sends back SSEs
func (s *SSEBroker) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	s.mutex.Lock()
	if s.lastMessage != nil {
		fmt.Fprintf(w, s.lastMessage.String())
		flusher.Flush()
	}
	s.mutex.Unlock()

	for msg := range s.send {
		fmt.Fprintf(w, msg)
		flusher.Flush()
	}
}

// SendSSE sends a Server Sent Event string to the HTTP handler to be pushed to
// the client
func (s *SSEBroker) SendSSE(msg *Message) {
	s.mutex.Lock()
	s.lastMessage = msg
	s.mutex.Unlock()
	s.send <- msg.String()
}
