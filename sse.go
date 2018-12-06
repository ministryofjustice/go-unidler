package main

import (
	"fmt"
	"log"
	"net/http"
)

type (
	// Message represents a Server Sent Event message
	Message struct {
		event string
		data  string
		id    string
		retry int
	}

	// Client represents a connected client (browser)
	Client struct {
		channel string
		send    chan string
	}

	// Channel is a named channel which can broadcast messages to a list of Clients
	Channel struct {
		name        string
		clients     map[*Client]bool
		lastMessage *Message
	}

	// SseSender provides a method to send a SSE event to a specified channel
	SseSender interface {
		SendSse(string, *Message)
	}

	// SseBroker represents the an server with a list of channels
	SseBroker struct {
		channels  map[string]*Channel
		addClient chan *Client
		delClient chan *Client
	}
)

func (m *Message) String() string {
	s := ""

	if m.id != "" {
		s = fmt.Sprintf("id: %s\n", m.id)
	}

	if m.event != "" {
		s = fmt.Sprintf("%sevent: %s\n", s, m.event)
	}

	if m.retry > 0 {
		s = fmt.Sprintf("%sretry: %d\n", s, m.retry)
	}

	if m.data != "" {
		s = fmt.Sprintf("%sdata: %s\n", s, m.data)
	}

	return fmt.Sprintf("%s\n", s)
}

func newClient(channel string) *Client {
	return &Client{
		channel: channel,
		send:    make(chan string),
	}
}

func newChannel(name string) *Channel {
	return &Channel{
		name:    name,
		clients: make(map[*Client]bool),
	}
}

func (ch *Channel) sendMessage(msg *Message) {
	ch.lastMessage = msg
	for c := range ch.clients {
		c.send <- msg.String()
	}
}

// NewSseBroker constructs a new SSE server and starts it running
func NewSseBroker() *SseBroker {
	s := &SseBroker{
		make(map[string]*Channel),
		make(chan *Client),
		make(chan *Client),
	}
	go s.dispatch()
	return s
}

// ServeHTTP receives HTTP requests from browsers and sends back SSEs
func (s *SseBroker) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.Printf("HTTP request received for %s", req.URL.Path)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// add the browser as a new client of the channel for the specified host
	host := req.Host
	q := req.URL.Query()
	if h, ok := q["host"]; ok {
		host = h[0]
	}
	c := newClient(host)
	s.addClient <- c
	log.Printf("client '%v' created", c)

	// remove the client if it closes the HTTP request
	closeNotify := w.(http.CloseNotifier).CloseNotify()
	go func() {
		<-closeNotify
		s.delClient <- c
		log.Printf("client '%v' closed request", c)
	}()

	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	for msg := range c.send {
		fmt.Fprintf(w, msg)
		flusher.Flush()
	}
}

// SendSse broadcasts a message to a channel (or all channels)
func (s *SseBroker) SendSse(channel string, msg *Message) {
	if channel == "" {
		log.Printf("broadcasting message to all channels: %s", msg)
		for _, ch := range s.channels {
			ch.sendMessage(msg)
		}
	} else if _, ok := s.channels[channel]; ok {
		log.Printf("message sent to channel '%s': %s", channel, msg)
		s.channels[channel].sendMessage(msg)
	} else {
		ch := newChannel(channel)
		s.channels[ch.name] = ch
		ch.lastMessage = msg
		log.Printf("message not sent because channel '%s' has no clients: %s", channel, msg)
	}
}

func (s *SseBroker) dispatch() {
	log.Print("SSE Broker started")

	for {
		select {
		case c := <-s.addClient:
			ch, exists := s.channels[c.channel]
			if !exists {
				ch = newChannel(c.channel)
				s.channels[ch.name] = ch
				log.Printf("created channel '%s'", ch.name)
			}
			ch.clients[c] = true
			log.Printf("added client to channel '%s'", ch.name)
			if ch.lastMessage != nil {
				log.Printf("sending last message in channel '%s'", ch.name)
				c.send <- ch.lastMessage.String()
			}

		case c := <-s.delClient:
			if ch, exists := s.channels[c.channel]; exists {
				close(c.send)
				delete(ch.clients, c)
				log.Printf("client removed from channel '%s'", ch.name)
			}
		}
	}
}
