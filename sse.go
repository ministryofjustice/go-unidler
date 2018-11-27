package main

import (
	"fmt"
	"net/http"
)

// Broker is responsible for keeping a list of which clients (browsers) are
// currently attached and broadcasting events (messages) to those clients.
type Broker struct {

	// Create a map of clients, the keys of the map are the channels
	// over which we can push messages to attached clients.  (The values
	// are empty structs which use no memory and are unused.)
	clients map[chan string]struct{}

	// Channel into which new clients can be pushed
	newClients chan chan string

	// Channel into which disconnected clients should be pushed
	defunctClients chan chan string

	// Channel into which messages are pushed to be broadcast out
	// to attached clients.
	messages chan string
}

// Start is a Broker method that starts a new goroutine.  It handles
// the addition and removal of clients, as well as the broadcasting
// of messages out to clients that are currently attached.
func (b *Broker) Start() {
	go func() {
		for {
			// Block until we receive from one of the
			// three following channels.
			select {

			case s := <-b.newClients:
				b.clients[s] = struct{}{}

			case s := <-b.defunctClients:
				delete(b.clients, s)
				close(s)

			case msg := <-b.messages:
				for s := range b.clients {
					s <- msg
				}
			}
		}
	}()
}

// ServeHTTP is a Broker method that handles and HTTP request at the "/events/" URL.
func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported!", http.StatusInternalServerError)
		return
	}

	messageChan := make(chan string)
	b.newClients <- messageChan

	// Listen to the closing of the http connection via the CloseNotifier
	notify := w.(http.CloseNotifier).CloseNotify()
	go func() {
		<-notify
		// Remove this client from the map of attached clients
		// when `EventHandler` exits.
		b.defunctClients <- messageChan
	}()

	// Set the headers related to event streaming.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {
		msg, open := <-messageChan

		if !open {
			break
		}

		fmt.Fprintf(w, "data: Message: %s\n\n", msg)

		f.Flush()
	}
}
