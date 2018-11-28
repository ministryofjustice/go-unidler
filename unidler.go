package main

import (
	"fmt"
	"log"
)

type (
	// StatusTracker affords the ability to report status changes
	StatusTracker interface {
		SetStatus(string)
	}

	// Unidler represents the unidler server
	Unidler struct {
		inProgress map[string]*UnidleTask
		k8s        KubernetesWrapper
		name       string
		namespace  string
		requests   chan string
		sse        *SseBroker
		taskEnd    chan string
	}

	// UnidleTask represents the progress of unidling an app
	UnidleTask struct {
		host    string
		unidler *Unidler
		Status  string
	}
)

// NewUnidler constructs an Unidler server and starts it
func NewUnidler(namespace string, name string, k8s KubernetesWrapper, sse *SseBroker) *Unidler {
	u := &Unidler{
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

// HandleRequests waits for unidle requests and ignores subsequent requests to
// unidle the same app if it is already in progress
func (u *Unidler) HandleRequests() {
	log.Print("Unidler started")

	for {
		select {
		// if a request to unidle a host is received and that host is not
		// currently being unidled, start unidling - it should be impossible to
		// receive a request for an already unidled app
		case host := <-u.requests:
			log.Printf("Unidle requested for '%s'", host)
			if _, exists := u.inProgress[host]; !exists {
				log.Print("Not currently in progress")
				task := &UnidleTask{
					host:    host,
					unidler: u,
				}

				// start unidling task concurrently
				go task.Run()

				u.inProgress[host] = task
			} else {
				log.Print("Already unidling")
			}

		// it should be impossible to receive multiple requests to end the same
		// unidling task, but just in case, check to see if the task exists
		// first. uses a channel instead of modifying inProgress from a
		// goroutine, as multiple different tasks could end at the same time.
		case host := <-u.taskEnd:
			log.Printf("Unidle end requested for '%s'", host)
			if _, exists := u.inProgress[host]; exists {
				log.Print("OK")
				delete(u.inProgress, host)
			} else {
				log.Print("Not in progress")
			}
		}
	}
}

// Unidle requests to unidle a specified hostname
func (u *Unidler) Unidle(host string) {
	u.requests <- host
}

// SendMessage sends an SSE message
func (u *Unidler) SendMessage(host string, msg string) {
	u.sse.SendMessage(host, msg)
}

// Run executes the steps to unidle an app
func (t *UnidleTask) Run() {
	log.Printf("Unidling '%s'...", t.host)

	app, err := NewApp(t.host, t.unidler.k8s, t)
	if err != nil {
		t.Fail(err)
		return
	}

	err = app.Unidle()

	if err != nil {
		t.Fail(err)
		return
	}

	t.End()
}

// SetStatus updates the status of the unidle task and sends an SSE
func (t *UnidleTask) SetStatus(status string) {
	t.Status = status
	t.unidler.SendMessage(t.host, status)
}

// Fail ends the task with a failure status
func (t *UnidleTask) Fail(err error) {
	msg := fmt.Sprintf("Failed unidling '%s': %s", t.host, err)
	log.Print(msg)
	t.SetStatus(msg)
	t.End()
}

// End sends a message to the unidler that this task has ended
func (t *UnidleTask) End() {
	t.unidler.taskEnd <- t.host
}
