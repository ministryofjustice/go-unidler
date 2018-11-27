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
	broker           *Broker
	ingressClassName string
	k8s              UnidlerK8sClient
	logger           *log.Logger
	unidlerIngress   *v1beta1.Ingress
)

func main() {
	logger = log.New(os.Stdout, "Unidler ", log.Ldate|log.Ltime|log.LUTC)
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
		logger.Fatalf("Couldn't determine HOME directory, is $HOME set?")
	}
	var err error
	k8s, err = NewK8sClient(filepath.Join(home, ".kube", "config"))
	if err != nil {
		logger.Fatalf("%s", err)
	}

	// start a Server Side Events broker
	broker = &Broker{
		make(map[chan string]struct{}),
		make(chan (chan string)),
		make(chan (chan string)),
		make(chan string),
	}
	broker.Start()

	// get the Unidler ingress once only
	unidlerIngress, err = k8s.Ingress(UnidlerNs, Unidler)
	if err != nil {
		logger.Fatalf("Can't find ingress '%s' (ns: '%s'): %s", Unidler, UnidlerNs, err)
	}

	http.HandleFunc("/", unidle)
	http.Handle("/events/", broker)
	http.HandleFunc("/healthz", healthCheck)

	logger.Printf("Starting server on port %s...", port)
	server := &http.Server{
		Addr:         port,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 2 * time.Minute,
		IdleTimeout:  2 * time.Minute,
	}
	if err := server.ListenAndServe(); err != nil {
		logger.Fatalf("Server didn't start: %s", err)
	}
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
		logger.Fatalf("Error parsing template: %s", err)
	}

	tmpl.Execute(w, req.Host)

	logger.Printf("Unidling '%s'...\n", req.Host)

	go func() {
		app := NewApp(req.Host, k8s)
		if app.err == nil {
			broker.messages <- fmt.Sprintf("App '%s' found", req.Host)
		}
		app.SetReplicas(1)
		if app.err == nil {
			broker.messages <- fmt.Sprint("Restoring replicas")
		}
		app.WaitForDeployment()
		if app.err == nil {
			broker.messages <- fmt.Sprint("Enabling ingress")
		}
		app.EnableIngress(ingressClassName)
		if app.err == nil {
			broker.messages <- fmt.Sprint("Removing from Unidler")
		}
		app.RemoveFromUnidlerIngress(unidlerIngress)
		if app.err == nil {
			broker.messages <- fmt.Sprint("Marking as unidled")
		}
		app.RemoveIdledMetadata()
		if app.err == nil {
			broker.messages <- fmt.Sprint("Done!")
			logger.Printf("'%s' unidled\n", req.Host)
		} else {
			broker.messages <- fmt.Sprintf("Failed: %s", err)
		}
	}()
}
