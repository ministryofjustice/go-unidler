package main

import (
	"fmt"
	"log"
)

type (
	// Unidler provides a method to unidle an app
	Unidler interface {
		EndTask(*UnidleTask)
		Unidle(string)
	}

	// UnidlerServer represents the unidler server
	UnidlerServer struct {
		inProgress map[string]*UnidleTask
		k8s        *KubernetesAPI
		name       string
		namespace  string
		requests   chan string
		sse        SseSender
		taskEnd    chan string
	}

	// UnidleTask represents the progress of unidling an app
	UnidleTask struct {
		app     *App
		host    string
		sse     SseSender
		unidler Unidler
	}
)

// NewUnidler constructs an Unidler server and starts it
func NewUnidler(namespace string, name string, k8s *KubernetesAPI, sse SseSender) *UnidlerServer {
	u := &UnidlerServer{
		inProgress: make(map[string]*UnidleTask),
		k8s:        k8s,
		name:       name,
		namespace:  namespace,
		requests:   make(chan string),
		sse:        sse,
		taskEnd:    make(chan string),
	}

	go u.HandleRequests()

	return u
}

// EndTask notifies the Unidler that a task has ended
func (u *UnidlerServer) EndTask(task *UnidleTask) {
	u.taskEnd <- task.host
}

// HandleRequests waits for unidle requests and ignores subsequent requests to
// unidle the same app if it is already in progress
func (u *UnidlerServer) HandleRequests() {
	log.Print("Unidler started")

	for {
		select {
		// if a request to unidle a host is received and that host is not
		// currently being unidled, start unidling - it should be impossible to
		// receive a request for an already unidled app
		case host := <-u.requests:
			u.startUnidling(host)

		// it should be impossible to receive multiple requests to end the same
		// unidling task, but just in case, check to see if the task exists
		// first. uses a channel instead of modifying inProgress from a
		// goroutine, as multiple different tasks could end at the same time.
		case host := <-u.taskEnd:
			u.stopUnidling(host)
		}
	}
}

func (u *UnidlerServer) unidleInProgress(host string) bool {
	_, exists := u.inProgress[host]
	return exists
}

func (u *UnidlerServer) startUnidling(host string) {
	if u.unidleInProgress(host) {
		return
	}

	app, err := NewApp(host, u.k8s)
	if err != nil {
		msg := &Message{
			event: "error",
			data:  fmt.Sprintf("%s", err),
		}
		log.Print(msg.data)
		u.sse.SendSse(host, msg)
		return
	}

	task := &UnidleTask{
		app:     app,
		host:    host,
		sse:     u.sse,
		unidler: u,
	}

	// start unidling task concurrently
	go task.Run()

	u.inProgress[host] = task
}

func (u *UnidlerServer) stopUnidling(host string) {
	if !u.unidleInProgress(host) {
		return
	}
	delete(u.inProgress, host)
}

// Unidle requests to unidle a specified hostname
func (u *UnidlerServer) Unidle(host string) {
	u.requests <- host
}

// Run executes the steps to unidle an app
func (t *UnidleTask) Run() {
	log.Printf("Unidling '%s'...", t.host)

	// listen for status changes and errors during app.Unidle()
	go func() {
		for {
			select {
			case status := <-t.app.status:
				msg := &Message{
					data: status,
				}
				if status == "Ready" {
					msg.event = "success"
					t.End()
				}
				t.sse.SendSse(t.host, msg)

			case err := <-t.app.err:
				t.Fail(err)
				break
			}
		}
	}()

	go t.app.Unidle()
}

// Fail ends the task with a failure status
func (t *UnidleTask) Fail(err error) {
	msg := &Message{
		event: "error",
		data:  fmt.Sprintf("%s", err),
	}
	log.Print(msg.data)
	t.sse.SendSse(t.host, msg)
	t.End()
}

// End sends a message to the unidler that this task has ended
func (t *UnidleTask) End() {
	t.unidler.EndTask(t)
}
