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

func (u *Unidler) log(msg string) {
	log.SetPrefix("unidler ")
	log.Print(msg)
}

func (u *Unidler) Run() {
	u.log = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)

	u.log.Printf("Unidling '%s'...", u.host)

	app, err := NewApp(u.host, u.k8s)
	if err != nil {
		u.Fail(err)
		return
	}

	u.sse.SendSSE(&Message{data: "Pending"})
	err = app.SetReplicas()
	if err != nil {
		u.Fail(err)
		return
	}
	u.sse.SendSSE(&Message{data: "Waiting for deployment"})
	err = app.WaitForDeployment()
	if err != nil {
		u.Fail(err)
		return
	}
	u.sse.SendSSE(&Message{data: "Enabling ingress"})
	err = app.EnableIngress(u.ingressClassName)
	if err != nil {
		u.Fail(err)
		return
	}
	u.sse.SendSSE(&Message{data: "Removing from unidler"})
	err = app.RemoveFromUnidlerIngress()
	if err != nil {
		u.Fail(err)
		return
	}
	u.sse.SendSSE(&Message{data: "Marking as unidled"})
	err = app.RemoveIdledMetadata()
	if err != nil {
		u.Fail(err)
		return
	}

	u.sse.SendSSE(&Message{
		event: "success",
		data:  "Ready",
	})
}

// Fail ends the unidle process with a failure status
func (u *Unidler) Fail(err error) {
	msg := &Message{
		event: "error",
		data:  err.Error(),
	}
	u.log.Print(msg.data)
	u.sse.SendSSE(msg)
}
