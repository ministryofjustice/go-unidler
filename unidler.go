package main

import (
	"fmt"
	"log"
)

type (
	// UnidleTask represents the progress of unidling an app
	UnidleTask struct {
		app  *App
		host string
		k8s  *KubernetesAPI
		sse  SseSender
	}
)

func (t *UnidleTask) run() {
	log.Printf("Unidling '%s'...", t.host)

	app, err := NewApp(t.host, t.k8s)
	if err != nil {
		t.Fail(err)
		return
	}

	t.sendStatus("Pending")
	err = app.SetReplicas(1)
	if err != nil {
		t.Fail(err)
		return
	}
	t.sendStatus("Waiting for deployment")
	err = app.WaitForDeployment()
	if err != nil {
		t.Fail(err)
		return
	}
	t.sendStatus("Enabling ingress")
	err = app.EnableIngress(ingressClassName)
	if err != nil {
		t.Fail(err)
		return
	}
	t.sendStatus("Removing from unidler")
	err = app.RemoveFromUnidlerIngress()
	if err != nil {
		t.Fail(err)
		return
	}
	t.sendStatus("Marking as unidled")
	err = app.RemoveIdledMetadata()
	if err != nil {
		t.Fail(err)
		return
	}

	t.sse.SendSse(t.host, &Message{
		event: "success",
		data:  "Ready",
	})
}

func (t *UnidleTask) sendStatus(data string) {
	t.sse.SendSse(t.host, &Message{data: data})
}

// Fail ends the task with a failure status
func (t *UnidleTask) Fail(err error) {
	msg := &Message{
		event: "error",
		data:  fmt.Sprintf("%s", err),
	}
	log.Print(msg.data)
	t.sse.SendSse(t.host, msg)
}
