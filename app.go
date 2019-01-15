package main

import (
	"fmt"
	"log"
	"os"
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
	// IngressClass is a standard kubernetes annotation name for the class of an
	// ingress
	IngressClass = "kubernetes.io/ingress.class"
)

type (
	// App is a Analytical Platform "app" consisting of a kubernetes
	// deployment, with a corresponding hostname and ingress
	App struct {
		deployment *v1.Deployment
		host       string
		ingress    *v1beta1.Ingress
		k8s        *KubernetesAPI
		logger     *log.Logger
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
	app.logger = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	app.deployment, err = k8s.Deployment(app.ingress)
	if err != nil {
		return nil, fmt.Errorf("failed getting app: %s", err)
	}
	return app, nil
}

func (a *App) log(msg string) {
	a.logger.Printf(
		"%s/%s: %s",
		a.ingress.Namespace,
		a.ingress.Name,
		msg,
	)
}

// SetReplicas updates an App's number of replicas to the specified number
func (a *App) SetReplicas() (err error) {
	replicas, err := a.numReplicasToRestore()
	if err != nil {
		return fmt.Errorf("failed setting replicas: %s", err)
	}

	a.deployment, err = a.k8s.PatchDeployment(
		a.deployment,
		Replace("/spec/replicas", replicas),
	)
	if err != nil {
		return fmt.Errorf("failed setting replicas to %d: %s", replicas, err)
	}

	a.log(fmt.Sprintf("Deployment replicas changed to %d", replicas))
	return nil
}

// numReplicasToRestore returns the number of replicas that were specified for
// an App Deployment, as parsed from the idled-at annotation
func (a *App) numReplicasToRestore() (int32, error) {
	idledAt, exists := a.deployment.Annotations[IdledAtAnnotation]
	if !exists {
		// app not idled, so return current number of replicas for a no-op
		return *a.deployment.Spec.Replicas, nil
	}

	// the idled-at annotation value is in the form <TIMESTAMP>,<NUM-REPLICAS>
	replicas, err := strconv.ParseInt(strings.Split(idledAt, ",")[1], 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed parsing num replicas to restore: %s", err)
	}
	return int32(replicas), nil
}

// EnableIngress updates the App's ingress to route requests to the Deployment
func (a *App) EnableIngress(ingressClassName string) (err error) {
	if ingressClassName == "" {
		ingressClassName = "nginx"
	}

	a.ingress, err = a.k8s.PatchIngress(
		a.ingress,
		Replace(
			JSONPointer("metadata", "annotations", IngressClass),
			ingressClassName,
		),
	)
	if err != nil {
		return fmt.Errorf("failed enabling ingress: %s", err)
	}

	a.log("Ingress is now enabled")
	return nil
}

// RemoveFromUnidlerIngress removes the App's hostname from the Unidler Ingress,
// preventing further requests from being handled by the Unidler
func (a *App) RemoveFromUnidlerIngress() error {
	unidlerIngress, err := a.k8s.Ingress(UnidlerNs, UnidlerName)
	if err != nil {
		return fmt.Errorf("couldn't find unidler ingress: %s", err)
	}

	filteredRules, _ := removeHostRule(a.host, unidlerIngress.Spec.Rules)
	_, err = a.k8s.PatchIngress(
		unidlerIngress,
		Replace("/spec/rules", filteredRules),
	)
	if err != nil {
		return fmt.Errorf("failed updating unidler ingress rules: %s", err)
	}

	a.log("Removed from unidler ingress")
	return nil
}

// removeHostRule returns a copy of the specified list of IngressRules, with any
// rules for the specified host removed
func removeHostRule(host string, rules []v1beta1.IngressRule) ([]v1beta1.IngressRule, bool) {
	filteredRules := []v1beta1.IngressRule{}
	found := false
	for _, rule := range rules {
		if rule.Host != host {
			filteredRules = append(filteredRules, rule)
		} else {
			found = true
		}
	}
	return filteredRules, found
}

// RemoveIdledMetadata removes the App's label and annotation which indicate its
// idled status, marking it as no longer idled
func (a *App) RemoveIdledMetadata() (err error) {
	a.deployment, err = a.k8s.PatchDeployment(
		a.deployment,
		Remove(JSONPointer("metadata", "annotations", IdledAtAnnotation)),
		Remove(JSONPointer("metadata", "labels", IdledLabel)),
	)
	if err != nil {
		return fmt.Errorf("failed removing idled metadata: %s", err)
	}

	a.log("Removed idled label/annotation from deployment")
	return nil
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
