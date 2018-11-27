package main

import (
	"fmt"
	"k8s.io/api/apps/v1"
	"k8s.io/api/extensions/v1beta1"
)

const (
	// IdledLabel is a metadata label which indicates a Deployment is idled
	IdledLabel = "mojanalytics.xyz/idled"
	// IdledAtAnnotation is a metadata annotation which indicates the time a
	// Deployment was idled and the number of replicas it had at that time,
	// separated by a semicolon, eg: "2018-11-26T17:27:34;2"
	IdledAtAnnotation = "mojanalytics.xyz/idled-at"
)

type (
	// App is a Analytical Platform "app" consisting of a kubernetes
	// deployment, with a corresponding hostname and ingress
	App struct {
		deployment *v1.Deployment
		err        error
		host       string
		ingress    *v1beta1.Ingress
		k8s        UnidlerK8sClient
	}
)

// NewApp constructs a new App and fetches the corresponding kubernetes ingress
// and deployment
func NewApp(host string, k8s UnidlerK8sClient) *App {
	app := App{
		host: host,
		k8s:  k8s,
	}
	app.ingress, app.err = k8s.IngressForHost(host)
	if app.err == nil {
		app.deployment, app.err = k8s.Deployment(app.ingress)
	}
	return &app
}

// SetReplicas updates an App's number of replicas to the specified number
func (a *App) SetReplicas(replicas int32) {
	if a.err != nil {
		return
	}

	oldReplicas := *a.deployment.Spec.Replicas
	ns, name := a.deployment.Namespace, a.deployment.Name
	a.deployment.Spec.Replicas = &replicas
	updated, err := a.k8s.UpdateDeployment(a.deployment)
	if err != nil {
		a.err = fmt.Errorf("failed setting replicas to %d: %s", replicas, err)
		return
	}

	a.deployment = updated
	logger.Printf("Deployment '%s' (ns: '%s') replicas changed from %d to %d", name, ns, oldReplicas, *a.deployment.Spec.Replicas)
}

// EnableIngress updates the App's ingress to route requests to the Deployment
func (a *App) EnableIngress(ingressClassName string) {
	if a.err != nil {
		return
	}

	if ingressClassName == "" {
		ingressClassName = "nginx"
	}

	name, ns := a.ingress.Name, a.ingress.Namespace

	a.ingress.Annotations["kubernetes.io/ingress.class"] = ingressClassName
	updated, err := a.k8s.UpdateIngress(a.ingress)
	if err != nil {
		a.err = fmt.Errorf("failed enabling ingress: %s", err)
		return
	}

	a.ingress = updated
	logger.Printf("Ingress '%s' (ns: '%s') is now enabled", name, ns)
}

// RemoveFromUnidlerIngress removes the App's hostname from the Unidler Ingress,
// preventing further requests from being handled by the Unidler
func (a *App) RemoveFromUnidlerIngress(unidlerIngress *v1beta1.Ingress) {
	if a.err != nil {
		return
	}

	// Remove rule for App's host
	newRules := []v1beta1.IngressRule{}
	for _, rule := range unidlerIngress.Spec.Rules {
		if rule.Host != a.host {
			newRules = append(newRules, rule)
		}
	}
	unidlerIngress.Spec.Rules = newRules

	_, err := a.k8s.UpdateIngress(unidlerIngress)
	if err != nil {
		a.err = fmt.Errorf("failed updating unidler ingress rules: %s", err)
		return
	}

	logger.Printf("Host '%s' removed from unidler ingress", a.host)
}

// RemoveIdledMetadata removes the App's label and annotation which indicate its
// idled status, marking it as no longer idled
func (a *App) RemoveIdledMetadata() {
	if a.err != nil {
		return
	}

	delete(a.deployment.Labels, IdledLabel)
	delete(a.deployment.Annotations, IdledAtAnnotation)

	updated, err := a.k8s.UpdateDeployment(a.deployment)
	if err != nil {
		a.err = fmt.Errorf("failed removing idled label/annotation: %s", err)
		return
	}

	a.deployment = updated

	logger.Printf("Removed idled label/annotation from deployment '%s' (ns: '%s')", a.deployment.Name, a.deployment.Namespace)
}

// WaitForDeployment blocks until the App's Deployment is ready to receive
// incoming requests
func (a *App) WaitForDeployment() {
	if a.err != nil {
		return
	}

	w, err := a.k8s.WatchDeployment(a.deployment)
	if err != nil {
		a.err = err
		return
	}

	for event := range w.ResultChan() {
		dep, ok := event.Object.(*v1.Deployment)
		if !ok {
			a.err = fmt.Errorf("unexpected event type: %+v", event.Object)
			return
		}

		if dep.Status.AvailableReplicas > 0 {
			logger.Printf("Deployment '%s' (ns: '%s') now has available replicas", dep.Name, dep.Namespace)
			break
		}
	}
}
