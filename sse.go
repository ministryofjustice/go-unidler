package main

import (
	"fmt"
	"log"
	"net/http"
)

type (
	Client struct {
		channel string
		send    chan string
	}

	Channel struct {
		name        string
		clients     map[*Client]bool
		lastMessage string
	}

	SseBroker struct {
		channels  map[string]*Channel
		addClient chan *Client
		delClient chan *Client
	}
)

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

func (ch *Channel) sendMessage(msg string) {
	ch.lastMessage = msg
	for c := range ch.clients {
		c.send <- msg
	}
}

func NewSseBroker() *SseBroker {
	s := &SseBroker{
		make(map[string]*Channel),
		make(chan *Client),
		make(chan *Client),
	}
	go s.dispatch()
	return s
}

func (s *SseBroker) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	c := newClient(req.Host)
	s.addClient <- c
	closeNotify := w.(http.CloseNotifier).CloseNotify()

	go func() {
		<-closeNotify
		s.delClient <- c
	}()

	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	for msg := range c.send {
		fmt.Fprintf(w, "data: %s\n\n", msg)
		flusher.Flush()
	}
}

func (s *SseBroker) SendMessage(channel string, msg string) {
	log.Printf("data: %s\n\n", msg)
	if len(channel) == 0 {
		log.Print("broadcasting message to all channels")
		for _, ch := range s.channels {
			ch.sendMessage(msg)
		}
	} else if _, ok := s.channels[channel]; ok {
		log.Printf("message sent to channel '%s'", channel)
		s.channels[channel].sendMessage(msg)
	} else {
		ch := newChannel(channel)
		s.channels[ch.name] = ch
		ch.lastMessage = msg
		log.Printf("message not sent because channel '%s' has no clients", channel)
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
			if ch.lastMessage != "" {
				log.Printf("sending last message in channel '%s'", ch.name)
				c.send <- ch.lastMessage
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
