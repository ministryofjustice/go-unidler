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
	// Unidler is the name of the kubernetes Unidler ingress
	Unidler = "unidler"
	// UnidlerNs is the namespace of the kubernetes Unidler ingress
	UnidlerNs = "default"
)

var (
	broker           *SseBroker
	ingressClassName string
	k8s              UnidlerK8sClient
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
	k8s, err = NewK8sClient(filepath.Join(home, ".kube", "config"))
	if err != nil {
		log.Fatalf("%s", err)
	}

	// start a Server Side Events broker
	broker = NewSseBroker()

	// get the Unidler ingress once only
	unidlerIngress, err = k8s.Ingress(UnidlerNs, Unidler)
	if err != nil {
		log.Fatalf("Can't find ingress '%s' (ns: '%s'): %s", Unidler, UnidlerNs, err)
	} else {
		log.Printf("Found unidler ingress")
	}

	http.HandleFunc("/", unidle)
	http.Handle("/events/", broker)
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
	fmt.Fprint(w, "Still OK")
}

func unidle(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatalf("Error parsing template: %s", err)
	}

	tmpl.Execute(w, req.Host)

	go func(host string) {
		log.Printf("Unidling '%s'...\n", host)

		broker.SendMessage(host, fmt.Sprintf("Fetching app '%s'", host))
		app := NewApp(host, k8s)
		if app.err == nil {
			broker.SendMessage(host, fmt.Sprintf("App '%s' found", host))
		}
		app.SetReplicas(1)
		if app.err == nil {
			broker.SendMessage(host, fmt.Sprint("Restoring replicas"))
		}
		app.WaitForDeployment()
		if app.err == nil {
			broker.SendMessage(host, fmt.Sprint("Enabling ingress"))
		}
		app.EnableIngress(ingressClassName)
		if app.err == nil {
			broker.SendMessage(host, fmt.Sprint("Removing from Unidler"))
		}
		app.RemoveFromUnidlerIngress(unidlerIngress)
		if app.err == nil {
			broker.SendMessage(host, fmt.Sprint("Marking as unidled"))
		}
		app.RemoveIdledMetadata()
		if app.err == nil {
			broker.SendMessage(host, fmt.Sprint("Done!"))
			log.Printf("'%s' unidled\n", host)
		} else {
			broker.SendMessage(host, fmt.Sprintf("Failed: %s", app.err))
		}
	}(req.Host)
}
