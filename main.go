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

var (
	ingressClassName string
	tmpl             *template.Template
)

type (
	Unidler struct {
		k8s *KubernetesAPI
		sse SseSender
	}
)

func main() {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = ":8080"
	}
	ingressClassName, ok = os.LookupEnv("INGRESS_CLASS_NAME")
	if !ok {
		ingressClassName = "nginx"
	}
	home, ok := os.LookupEnv("HOME")
	if !ok {
		log.Fatalf("Couldn't determine HOME directory, is $HOME set?")
	}
	var err error
	k8s, err := NewKubernetesAPI(filepath.Join(home, ".kube", "config"))
	if err != nil {
		log.Fatalf("%s", err)
	}

	// parse HTML template
	tmpl, err = template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatalf("Error parsing template: %s", err)
	}

	// start a Server Side Events broker
	sse := NewSseBroker()

	u := &Unidler{
		k8s: k8s,
		sse: sse,
	}

	http.HandleFunc("/", u.unidle)
	http.Handle("/events/", sse)
	http.HandleFunc("/healthz", healthCheck)

	log.Printf("Starting server on port %s...", port)
	server := &http.Server{
		Addr:         port,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 2 * time.Minute,
		IdleTimeout:  2 * time.Minute,
	}
	log.Fatal(server.ListenAndServe())
}

func healthCheck(w http.ResponseWriter, req *http.Request) {
	log.Printf("HTTP request received for %s", req.URL.Path)
	fmt.Fprint(w, "Still OK")
}

func (u *Unidler) unidle(w http.ResponseWriter, req *http.Request) {
	host := getHost(req)
	tmpl.Execute(w, host)

	task := &UnidleTask{host: host, k8s: u.k8s, sse: u.sse}
	go task.run()
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
