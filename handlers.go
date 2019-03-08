package main

import (
	"fmt"
	"net/http"
)

// StreamingResponseWriter is a convenience interface
// representing a streaming HTTP response
type StreamingResponseWriter interface {
	http.ResponseWriter
	http.Flusher
}

// Index renders the index page
func indexHandler(w http.ResponseWriter, req *http.Request) {
	indexTemplates.ExecuteTemplate(w, "layout", req.Host)
}

// Unidles an app and sends status updates to the client as SSEs
func eventsHandler(w http.ResponseWriter, req *http.Request) {
	s, ok := w.(StreamingResponseWriter)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	w.WriteHeader(http.StatusOK)
	s.Flush()

	sendMessage(s, "Pending")

	app, err := NewApp(req.Host)
	if err != nil {
		sendError(s, err)
		return
	}

	sendMessage(s, "Restoring app")

	err = app.SetReplicas()
	if err != nil {
		sendError(s, err)
		return
	}

	err = app.WaitForDeployment()
	if err != nil {
		sendError(s, err)
		return
	}

	err = app.RemoveIdledMetadata()
	if err != nil {
		sendError(s, err)
		return
	}

	err = app.RedirectService()
	if err != nil {
		sendError(s, err)
		return
	}

	sendEvent(s, &Message{
		event: "success",
		data:  "Ready",
	})
}

func healthCheckHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, "Still OK")
}
