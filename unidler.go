package main

import (
	"html/template"
	"log"
	"os"
)

type (
	// Unidler represents the progress of unidling an app
	Unidler struct {
		app              *App
		host             string
		ingressClassName string
		k8s              *KubernetesAPI
		log              *log.Logger
		sse              SSESender
		tmpl             *template.Template
	}
)

func (u *Unidler) sendSSE(msg string) {
	u.sse.SendSSE(&Message{
		data:  msg,
		group: u.host,
	})
}

// Run executes the steps to unidle the specified app
func (u *Unidler) Run() {
	u.log = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)

	u.log.Printf("Unidling '%s'...", u.host)

	app, err := NewApp(u.host, u.k8s)
	if err != nil {
		u.Fail(err)
		return
	}

	u.sendSSE("Pending")
	err = app.SetReplicas()
	if err != nil {
		u.Fail(err)
		return
	}
	u.sendSSE("Waiting for deployment")
	err = app.WaitForDeployment()
	if err != nil {
		u.Fail(err)
		return
	}
	u.sendSSE("Enabling ingress")
	err = app.EnableIngress(u.ingressClassName)
	if err != nil {
		u.Fail(err)
		return
	}
	u.sendSSE("Removing from unidler")
	err = app.RemoveFromUnidlerIngress()
	if err != nil {
		u.Fail(err)
		return
	}
	u.sendSSE("Marking as unidled")
	err = app.RemoveIdledMetadata()
	if err != nil {
		u.Fail(err)
		return
	}

	u.sse.SendSSE(&Message{
		event: "success",
		data:  "Ready",
		group: u.host,
	})
}

// Fail ends the unidle process with a failure status
func (u *Unidler) Fail(err error) {
	msg := &Message{
		event: "error",
		data:  err.Error(),
		group: u.host,
	}
	u.log.Print(msg.data)
	u.sse.SendSSE(msg)
}
