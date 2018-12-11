package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"k8s.io/api/extensions/v1beta1"
)

const (
	// UnidlerName is the name of the kubernetes Unidler ingress
	UnidlerName = "unidler"
	// UnidlerNs is the namespace of the kubernetes Unidler ingress
	UnidlerNs = "default"
)

var (
	sse              *SseBroker
	ingressClassName string
	k8s              *KubernetesAPI
	tmpl             *template.Template
	unidlerIngress   *v1beta1.Ingress
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
	k8s, err = NewKubernetesAPI(filepath.Join(home, ".kube", "config"))
	if err != nil {
		log.Fatalf("%s", err)
	}

	// parse HTML template
	tmpl, err = template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatalf("Error parsing template: %s", err)
	}

	// start a Server Side Events broker
	sse = NewSseBroker()

	http.HandleFunc("/", unidle)
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

func unidle(w http.ResponseWriter, req *http.Request) {
	host := getHost(req)
	tmpl.Execute(w, host)

	task := &UnidleTask{host: host}
	go task.Run(k8s)
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
