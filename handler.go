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
	host := r.Host

	h.Config.Logger.Printf("Unidling '%s'...\n", host)

	app := NewApp(host, h.Config)
	err := app.Unidle()
	if err != nil {
		h.Config.Logger.Printf("Error while unidling '%s': %s\n", host, err)

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error while unidling, please contact one of the developers\n")

		return
	}

	h.Config.Logger.Printf("'%s' unidled\n", host)
	http.Redirect(w, r, r.URL.String(), http.StatusTemporaryRedirect)
}
