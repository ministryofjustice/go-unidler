package main

import (
	"fmt"
	"log"
	"strings"

	"k8s.io/api/apps/v1"
	"k8s.io/api/extensions/v1beta1"
)

const (
	// IdledLabel is a metadata label which indicates a Deployment is idled.
	IdledLabel = "mojanalytics.xyz/idled"
	// IdledAtAnnotation is a metadata annotation which indicates the time a
	// Deployment was idled and the number of replicas it had at that time,
	// separated by a semicolon, eg: "2018-11-26T17:27:34;2".
	IdledAtAnnotation = "mojanalytics.xyz/idled-at"
)

type (
	// App is a Analytical Platform "app" consisting of a kubernetes
	// deployment, with a corresponding hostname and ingress
	App struct {
		deployment *v1.Deployment
		host       string
		ingress    *v1beta1.Ingress
		k8s        KubernetesWrapper
		name       string
		namespace  string
		status     StatusTracker
	}
)

// NewApp constructs a new App and fetches the corresponding kubernetes ingress
// and deployment
func NewApp(host string, k8s KubernetesWrapper, s StatusTracker) (app *App, err error) {
	app = &App{
		host:   host,
		k8s:    k8s,
		status: s,
	}
	app.ingress, err = k8s.IngressForHost(host)
	if err != nil {
		return nil, err
	}
	app.name = app.ingress.Name
	app.namespace = app.ingress.Namespace
	app.deployment, err = k8s.Deployment(app.ingress)
	if err != nil {
		return nil, err
	}
	return app, nil
}

// Unidle performs the actions to unidle an app
func (a *App) Unidle() (err error) {
	a.setStatus("Pending")

	if err = a.SetReplicas(1); err != nil {
		return
	}
	if err = a.WaitForDeployment(); err != nil {
		return
	}
	if err = a.EnableIngress(ingressClassName); err != nil {
		return
	}
	if err = a.RemoveFromUnidlerIngress(); err != nil {
		return
	}
	if err = a.RemoveIdledMetadata(); err != nil {
		return
	}
	a.setStatus("Ready")
	return
}

func (a *App) log(msg string) {
	log.Printf("Unidling '%s' (ns: '%s'): %s", a.name, a.namespace, msg)
}

func (a *App) setStatus(s string) {
	a.status.SetStatus(s)
}

// SetReplicas updates an App's number of replicas to the specified number
func (a *App) SetReplicas(replicas int32) error {
	current := *a.deployment.Spec.Replicas

	if current != replicas {
		a.setStatus("Restoring replicas")

		a.deployment.Spec.Replicas = &replicas
		_, err := a.k8s.UpdateDeployment(a.deployment)
		if err != nil {
			return fmt.Errorf("failed setting replicas to %d: %s", replicas, err)
		}

		a.log(fmt.Sprintf("Deployment replicas changed to %d", replicas))
	} else {
		a.log(fmt.Sprintf("Deployment replicas already %d", replicas))
	}

	return nil
}

// EnableIngress updates the App's ingress to route requests to the Deployment
func (a *App) EnableIngress(ingressClassName string) error {
	ingressClass := "kubernetes.io/ingress.class"
	if ingressClassName == "" {
		ingressClassName = "nginx"
	}

	current := a.ingress.Annotations[ingressClass]

	if current != ingressClassName {

		a.setStatus("Enabling ingress")
		a.ingress.Annotations[ingressClass] = ingressClassName
		if _, err := a.k8s.UpdateIngress(a.ingress); err != nil {
			return fmt.Errorf("failed enabling ingress: %s", err)
		}

		a.log("Ingress is now enabled")
	} else {
		a.log(fmt.Sprintf("Ingress already '%s'", ingressClassName))
	}
	return nil
}

// RemoveFromUnidlerIngress removes the App's hostname from the Unidler Ingress,
// preventing further requests from being handled by the Unidler
func (a *App) RemoveFromUnidlerIngress() error {
	unidlerIngress, err := a.k8s.Ingress(UnidlerNs, UnidlerName)
	if err != nil {
		return fmt.Errorf("couldn't find unidler ingress: %s", err)
	}

	// Remove rule for App's host
	newRules := []v1beta1.IngressRule{}
	found := false
	for _, rule := range unidlerIngress.Spec.Rules {
		if rule.Host != a.host {
			newRules = append(newRules, rule)
		} else {
			found = true
		}
	}
	unidlerIngress.Spec.Rules = newRules

	if found {
		a.setStatus("Removing from Unidler")

		if _, err := a.k8s.UpdateIngress(unidlerIngress); err != nil {
			return fmt.Errorf("failed updating unidler ingress rules: %s", err)
		}

		a.log("Removed from unidler ingress")
	} else {
		a.log("Not in unidler ingress")
	}
	return nil
}

// RemoveIdledMetadata removes the App's label and annotation which indicate its
// idled status, marking it as no longer idled
func (a *App) RemoveIdledMetadata() error {
	_, labelExists := a.deployment.Labels[IdledLabel]
	_, annotationExists := a.deployment.Annotations[IdledAtAnnotation]

	if labelExists || annotationExists {
		a.setStatus("Marking as unidled")

		patch := fmt.Sprintf(`[
			{"op": "remove", "path": "/metadata/labels/%s"},
			{"op": "remove", "path": "/metadata/annotations/%s"}
		]`, jsonPatchEscape(IdledLabel), jsonPatchEscape(IdledAtAnnotation))
		if _, err := a.k8s.PatchDeployment(a.deployment, patch); err != nil {
			return fmt.Errorf("failed removing idled label/annotation: %s", err)
		}

		a.log("Removed idled label/annotation from deployment")
	} else {
		a.log("Deployment not marked as idled")
	}
	return nil
}

// JSON patch requires "~" and "/" characters to be escaped as "~0" and "~1"
// respectively. See http://jsonpatch.com/#json-pointer
func jsonPatchEscape(s string) string {
	return strings.Replace(strings.Replace(s, "~", "~0", -1), "/", "~1", -1)
}

// WaitForDeployment blocks until the App's Deployment is ready to receive
// incoming requests
func (a *App) WaitForDeployment() error {
	w, err := a.k8s.WatchDeployment(a.deployment)
	if err != nil {
		return err
	}

	for event := range w.ResultChan() {
		dep, ok := event.Object.(*v1.Deployment)
		if !ok {
			return fmt.Errorf("unexpected event type: %+v", event.Object)
		}

		if dep.Status.AvailableReplicas > 0 {
			break
		}
	}

	a.log("Deployment has available replicas")
	return nil
}
