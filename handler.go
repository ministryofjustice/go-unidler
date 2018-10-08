package main

import (
	"fmt"
	"net/http"
)

type Handlers struct {
	Config *Config
}

func NewHandlers(config *Config) *Handlers {
	return &Handlers{
		Config: config,
	}
}

func (h *Handlers) Unidler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "TODO: Unidling...")
}
