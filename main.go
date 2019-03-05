package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	k8s "k8s.io/client-go/kubernetes"
)

const (
	// IdledLabel is a metadata label which indicates a Deployment is idled.
	IdledLabel = "mojanalytics.xyz/idled"
	// IdledAtAnnotation is a metadata annotation which indicates the time a
	// Deployment was idled and the number of replicas it had at that time,
	// separated by a semicolon, eg: "2018-11-26T17:27:34;2".
	IdledAtAnnotation = "mojanalytics.xyz/idled-at"
	// UnidlerName is the name of the kubernetes Unidler ingress
	UnidlerName = "unidler"
	// UnidlerNs is the namespace of the kubernetes Unidler ingress
	UnidlerNs = "default"
)

type (
	// Context is a context holder for the unidle handler
	Context struct {
		k8s  k8s.Interface
		tmpl *template.Template
	}

	// Message represents a Server Sent Event message
	Message struct {
		data  string
		event string
		group string
		id    string
		retry int
	}

	// StreamingResponseWriter is a convenience interface representing a streaming
	// HTTP response
	StreamingResponseWriter interface {
		http.ResponseWriter
		http.Flusher
	}
)

var (
	logger *log.Logger
)

func main() {
	logger = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)

	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = ":8080"
	}
	home, ok := os.LookupEnv("HOME")
	if !ok {
		logger.Fatalf("Couldn't determine HOME directory, is $HOME set?")
	}

	k8s, err := KubernetesClient(filepath.Join(home, ".kube", "config"))
	if err != nil {
		log.Fatalf("%s", err)
	}

	// parse HTML template
	tmpl, err := template.New("").ParseFiles(
		"templates/content.html",
		"templates/javascript.js",
		"templates/throbber.html",
		"templates/layout.html",
	)
	if err != nil {
		logger.Fatalf("Error parsing template: %s", err)
	}

	ctx := &Context{
		k8s:  k8s,
		tmpl: tmpl,
	}

	http.HandleFunc("/", ctx.Index)
	http.HandleFunc("/events/", ctx.Unidle)
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

func getHost(req *http.Request) string {
	host := req.Host

	// for testing purposes, allow host to be supplied as a URL parameter
	q := req.URL.Query()
	if h, ok := q["host"]; ok {
		host = h[0]
	}

	return host
}

// Index renders the index page
func (c *Context) Index(w http.ResponseWriter, req *http.Request) {
	c.tmpl.ExecuteTemplate(w, "layout", getHost(req))
}

// Unidle unidles an app and sends status updates to the client as SSEs
func (c *Context) Unidle(w http.ResponseWriter, req *http.Request) {
	s, ok := w.(StreamingResponseWriter)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	w.WriteHeader(http.StatusOK)
	s.Flush()

	sendMessage(s, "Pending")

	app, err := NewApp(getHost(req), c.k8s)
	if err != nil {
		sendError(s, err)
		return
	}

	sendMessage(s, "Restoring app")

	err = app.SetReplicas()
	if err != nil {
		sendError(s, err)
		return
	}

	err = app.WaitForDeployment()
	if err != nil {
		sendError(s, err)
		return
	}

	err = app.RemoveIdledMetadata()
	if err != nil {
		sendError(s, err)
		return
	}

	err = app.RemoveIdledMetadata()
	if err != nil {
		sendError(s, err)
		return
	}

	sendEvent(s, &Message{
		event: "success",
		data:  "Ready",
	})
}

func sendEvent(s StreamingResponseWriter, m *Message) {
	fmt.Fprintf(s, m.String())
	s.Flush()
}

func sendMessage(s StreamingResponseWriter, msg string) {
	sendEvent(s, &Message{
		data: msg,
	})
}

func sendError(s StreamingResponseWriter, err error) {
	sendEvent(s, &Message{
		event: "error",
		data:  err.Error(),
	})
}

func (m *Message) String() string {
	return fmt.Sprintf(`id: %s
retry: %d
event: %s
data: %s

`, m.id, m.retry, m.event, m.data)
}
