package main

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	IDLED_LABEL    = "mojanalytics.xyz/idled"
	IDLED_AT_LABEL = "mojanalytics.xyz/idled-at"
	UNIDLER        = "unidler"
	UNIDLER_NS     = "default"
)

type App struct {
	Host   string
	Config *Config
}

func NewApp(host string, config *Config) *App {
	return &App{
		Host:   host,
		Config: config,
	}
}

func (a *App) Unidle() error {
	err := a.setReplicas(1)
	if err != nil {
		return err
	}

	for !a.isRunning() {
		time.Sleep(25 * time.Millisecond)
	}

	err = a.enableIngress()
	if err != nil {
		return err
	}

	// err = a.removeIdledMetadata()
	// if err != nil {
	// 	return err
	// }

	return nil
}

// FYI: k8s has some kind of watch
func (a *App) isRunning() bool {
	// TODO
	return false
}

func (a *App) getReplicasBeforeIdled() int {
	// TODO
	return 1
}

func (a *App) removeIdledLabels() error {
	// TODO
	return nil
}

func (a *App) setReplicas(replicas int) error {
	pods, _ := a.Config.K8s.CoreV1().Pods("").List(metav1.ListOptions{})
	fmt.Println("PODS = %+v", pods)

	return nil
}

func (a *App) enableIngress() error {
	// TODO
	return nil
}
