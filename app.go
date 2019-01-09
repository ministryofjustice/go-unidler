package main

import (
	"fmt"
	"log"
	"strconv"
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
		k8s        *KubernetesAPI
		name       string
		namespace  string
	}
)

// NewApp constructs a new App and fetches the corresponding kubernetes ingress
// and deployment
func NewApp(host string, k8s *KubernetesAPI) (app *App, err error) {
	app = &App{
		host: host,
		k8s:  k8s,
	}
	app.ingress, err = k8s.IngressForHost(host)
	if err != nil {
		return nil, fmt.Errorf("failed getting app: %s", err)
	}
	app.name = app.ingress.Name
	app.namespace = app.ingress.Namespace
	app.deployment, err = k8s.Deployment(app.ingress)
	if err != nil {
		return nil, fmt.Errorf("failed getting app: %s", err)
	}
	return app, nil
}

func (a *App) log(msg string) {
	log.SetPrefix("app ")
	log.Printf("Unidling '%s' (ns: '%s'): %s", a.name, a.namespace, msg)
}

// SetReplicas updates an App's number of replicas to the specified number
func (a *App) SetReplicas() error {
	current := *a.deployment.Spec.Replicas
	restore, err := strconv.ParseInt(
		strings.Split(a.deployment.Annotations[IdledAtAnnotation], ",")[1],
		10,
		32,
	)
	if err != nil {
		return fmt.Errorf("failed parsing number of replicas to restore: %s", err)
	}

	replicas := int32(restore)
	if current != replicas {
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

	// re-retrieve the ingress to avoid "object has been modified" error
	var err error
	a.ingress, err = a.k8s.IngressForHost(a.host)
	if err != nil {
		return fmt.Errorf("failed getting app ingress: %s", err)
	}

	current := a.ingress.Annotations[ingressClass]

	if current != ingressClassName {
		a.ingress.Annotations[ingressClass] = ingressClassName
		_, err := a.k8s.UpdateIngress(a.ingress)
		if err != nil {
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
	ing, err := a.k8s.Ingress(UnidlerNs, UnidlerName)
	if err != nil {
		return fmt.Errorf("couldn't find unidler ingress: %s", err)
	}

	// Remove rule for App's host
	var found bool
	ing.Spec.Rules, found = removeHostRule(a.host, ing.Spec.Rules)

	if found {
		_, err := a.k8s.UpdateIngress(ing)
		if err != nil {
			return fmt.Errorf("failed updating unidler ingress rules: %s", err)
		}

		a.log("Removed from unidler ingress")
	} else {
		a.log("Not in unidler ingress")
	}
	return nil
}

func removeHostRule(host string, rules []v1beta1.IngressRule) ([]v1beta1.IngressRule, bool) {
	newRules := []v1beta1.IngressRule{}
	found := false
	for _, rule := range rules {
		if rule.Host != host {
			newRules = append(newRules, rule)
		} else {
			found = true
		}
	}
	return newRules, found
}

// RemoveIdledMetadata removes the App's label and annotation which indicate its
// idled status, marking it as no longer idled
func (a *App) RemoveIdledMetadata() error {
	// re-retrieve the deployment to avoid "object has been modified" error
	var err error
	a.deployment, err = a.k8s.Deployment(a.ingress)
	if err != nil {
		return fmt.Errorf("failed removing idled metadata: %s", err)
	}

	_, labelExists := a.deployment.Labels[IdledLabel]
	_, annotationExists := a.deployment.Annotations[IdledAtAnnotation]

	if labelExists || annotationExists {
		delete(a.deployment.Annotations, IdledAtAnnotation)
		delete(a.deployment.Labels, IdledLabel)
		_, err := a.k8s.UpdateDeployment(a.deployment)
		if err != nil {
			return fmt.Errorf("failed removing idled metadata: %s", err)
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
		return fmt.Errorf("failed watching deployment: %s", err)
	}

	for event := range w.ResultChan() {
		dep, ok := event.Object.(*v1.Deployment)
		if !ok {
			return fmt.Errorf("failed watching deployment: unexpected event type: %+v", event.Object)
		}

		if dep.Status.AvailableReplicas > 0 {
			break
		}
	}

	a.log("Deployment has available replicas")
	return nil
}
