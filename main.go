package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	// UnidlerName is the name of the kubernetes Unidler ingress
	UnidlerName = "unidler"
	// UnidlerNs is the namespace of the kubernetes Unidler ingress
	UnidlerNs = "default"
)

var logger *log.Logger

func main() {
	logger = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)

	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = ":8080"
	}
	ingressClassName, ok := os.LookupEnv("INGRESS_CLASS_NAME")
	if !ok {
		ingressClassName = "nginx"
	}
	home, ok := os.LookupEnv("HOME")
	if !ok {
		logger.Fatalf("Couldn't determine HOME directory, is $HOME set?")
	}
	var err error
	k8s, err := NewKubernetesAPI(filepath.Join(home, ".kube", "config"))
	if err != nil {
		log.Fatalf("%s", err)
	}

	// parse HTML template
	tmpl, err := template.New("").ParseFiles(
		"templates/index.html",
		"templates/index.js",
		"templates/throbber.html",
		"templates/base.html",
	)
	if err != nil {
		logger.Fatalf("Error parsing template: %s", err)
	}

	// start a Server Side Events broker
	sse := NewSSEBroker()

	u := &Unidler{
		ingressClassName: ingressClassName,
		k8s:              k8s,
		sse:              sse,
		tmpl:             tmpl,
	}

	http.HandleFunc("/", u.unidle)
	http.Handle("/events/", sse)
	http.HandleFunc("/healthz", healthCheck)

	logger.Printf("Starting server on port %s...", port)
	server := &http.Server{
		Addr:         port,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 2 * time.Minute,
		IdleTimeout:  2 * time.Minute,
	}
	log.Fatal(server.ListenAndServe())
}

func healthCheck(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, "Still OK")
}

func (u *Unidler) unidle(w http.ResponseWriter, req *http.Request) {
	u.host = getHost(req)
	u.tmpl.ExecuteTemplate(w, "base", u.host)
	go u.Run()
}

func getHost(req *http.Request) string {
	host := req.Host

	// for testing purposes, allow host to be supplied as a URL parameter
	q := req.URL.Query()
	if h, ok := q["host"]; ok {
		host = h[0]
	}

	return host
}
