package main

import (
	"net/http"
	"time"
)

// Returns a HTTP server which times out
//
// Default Go HTTP server (`http.ListenAndServe`) doesn't set timeouts
// See: https://blog.cloudflare.com/exposing-go-on-the-internet/
func NewServer(conf *Config) *http.Server {
	return &http.Server{
		Addr:         conf.Port,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 2 * time.Minute,
		IdleTimeout:  2 * time.Minute,
	}
}
