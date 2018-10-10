package main

import (
	"net/http"
)

func main() {
	config := LoadConfig()

	h := NewHandlers(config)
	http.HandleFunc("/", h.Unidler)
	http.HandleFunc("/healthz", h.Healthz)

	config.Logger.Printf("Starting server on port %s...", config.Port)
	srv := NewServer(config)
	err := srv.ListenAndServe()
	if err != nil {
		config.Logger.Panicf("Server didn't start: %s", err)
	}
}
