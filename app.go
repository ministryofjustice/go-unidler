package main

import (
	"fmt"
	"time"

	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	IDLED_LABEL    = "mojanalytics.xyz/idled"
	IDLED_AT_LABEL = "mojanalytics.xyz/idled-at"
	UNIDLER        = "unidler"
	UNIDLER_NS     = "default"
)

type App struct {
	Host    string
	Config  *Config
	ingress *v1beta1.Ingress
}

func NewApp(host string, config *Config) *App {
	return &App{
		Host:   host,
		Config: config,
	}
}

func (a *App) Unidle() error {
	err := a.getIngress()
	if err != nil {
		return err
	}

	err = a.setReplicas(1)
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

func (a *App) getIngress() error {
	opts := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name!=%s", UNIDLER),
	}

	// NOTE: can't filter by spec.rules[0].host
	list, err := a.Config.K8s.ExtensionsV1beta1().Ingresses("").List(opts)
	if err != nil {
		return err
	}

	for _, ing := range list.Items {
		if ing.Spec.Rules[0].Host == a.Host {
			a.Config.Logger.Printf("Ingress for host found: %s (ns: %s)\n", ing.Name, ing.Namespace)
			a.ingress = &ing
			return nil
		}
	}

	return fmt.Errorf("Cannot find ingress for host '%s'", a.Host)
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
	// pods, _ := a.Config.K8s.CoreV1().Pods("").List(metav1.ListOptions{})
	// fmt.Println("PODS = %+v", pods)

	return nil
}

func (a *App) enableIngress() error {
	// TODO
	return nil
}
