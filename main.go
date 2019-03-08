package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	k8s "k8s.io/client-go/kubernetes"
)

var (
	logger         *log.Logger
	k8sClient      k8s.Interface
	indexTemplates *template.Template
)

func init() {
	var err error

	// parse HTML template
	indexTemplates, err = template.New("").ParseFiles(
		"templates/content.html",
		"templates/javascript.js",
		"templates/throbber.html",
		"templates/layout.html",
	)
	if err != nil {
		logger.Fatalf("Error parsing template: %s", err)
	}
}

func main() {
	var err error

	logger = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)

	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = ":8080"
	}
	home, ok := os.LookupEnv("HOME")
	if !ok {
		logger.Fatalf("Couldn't determine HOME directory, is $HOME set?")
	}

	k8sClient, err = KubernetesClient(filepath.Join(home, ".kube", "config"))
	if err != nil {
		log.Fatalf("%s", err)
	}

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/events/", eventsHandler)
	http.HandleFunc("/healthz", healthzHandler)

	logger.Printf("Starting server on port %s...", port)
	server := &http.Server{
		Addr:         port,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 2 * time.Minute,
		IdleTimeout:  2 * time.Minute,
	}
	log.Fatal(server.ListenAndServe())
}
