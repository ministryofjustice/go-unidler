package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

type (
	// Message represents a Server Sent Event message
	Message struct {
		data  string
		event string
		group string
		id    string
		retry int
	}

	// SSESender provides a method to send a SSE event to a specified channel
	SSESender interface {
		SendSSE(*Message)
	}

	// Client represents an HTTP client that receives events
	Client struct {
		group string
		send  chan *Message
	}

	// Group is a Group of Clients
	Group struct {
		clients     map[*Client]bool
		lastMessage *Message
		name        string
	}

	// SSEBroker represents the server with a list of channels
	SSEBroker struct {
		addClient    chan *Client
		groups       map[string]*Group
		log          *log.Logger
		removeClient chan *Client
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
	broker := &SSEBroker{
		addClient:    make(chan *Client),
		groups:       make(map[string]*Group),
		log:          log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile),
		removeClient: make(chan *Client),
	}

	broker.log.Print("Starting SSE broker...")
	go broker.dispatch()

	return broker
}

// NewClient constructs a new SSE client
func NewClient(host string) *Client {
	return &Client{
		group: host,
		send:  make(chan *Message),
	}
}

// NewGroup constructs a new SSE client group
func NewGroup(host string) *Group {
	return &Group{
		name:    host,
		clients: make(map[*Client]bool),
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

	client := NewClient(getHost(req))
	s.addClient <- client

	closeNotify := w.(http.CloseNotifier).CloseNotify()
	go func() {
		<-closeNotify
		s.removeClient <- client
	}()

	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	for msg := range client.send {
		fmt.Fprintf(w, msg.String())
		flusher.Flush()
	}
}

// SendSSE sends a Server Sent Event string to the HTTP handler to be pushed to
// the client
func (s *SSEBroker) SendSSE(msg *Message) {
	group := s.getOrCreateGroup(msg.group)
	group.lastMessage = msg

	if !group.isEmpty() {
		for client, open := range group.clients {
			if open {
				client.send <- msg
			}
		}
	}
}

func (s *SSEBroker) dispatch() {
	for {
		select {
		case client := <-s.addClient:
			s.add(client)

		case client := <-s.removeClient:
			s.remove(client)
		}
	}
}

func (s *SSEBroker) add(client *Client) {
	group := s.getOrCreateGroup(client.group)
	group.add(client)
	s.log.Printf("Added client to group %s", group.name)

	if group.lastMessage != nil {
		client.send <- group.lastMessage
	}
}

func (s *SSEBroker) getOrCreateGroup(name string) *Group {
	group, exists := s.groups[name]
	if !exists {
		group = NewGroup(name)
		s.groups[name] = group
		s.log.Printf("Created group %s", name)
	}
	return group
}

func (s *SSEBroker) remove(client *Client) {
	group, exists := s.groups[client.group]
	if !exists {
		s.log.Printf("Group %s does not exist", group.name)
		return
	}
	group.remove(client)
	s.log.Printf("Removed client from group %s", group.name)
	if group.isEmpty() {
		delete(s.groups, group.name)
		s.log.Printf("Removed empty group %s", group.name)
	}
}

func (g *Group) add(client *Client) {
	g.clients[client] = true
}

func (g *Group) remove(client *Client) {
	g.clients[client] = false
	close(client.send)
	delete(g.clients, client)
}

func (g *Group) isEmpty() bool {
	return len(g.clients) == 0
}
