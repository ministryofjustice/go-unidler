package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"k8s.io/api/apps/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
		deployment *Deployment
		host       string
		ingress    *Ingress
		k8s        kubernetes.Interface
		logger     *log.Logger
	}
)

// NewApp constructs a new App and fetches the corresponding kubernetes ingress
// and deployment
func NewApp(host string, k kubernetes.Interface) (app *App, err error) {
	app = &App{
		host:   host,
		k8s:    k,
		logger: log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile),
	}
	app.ingress, err = app.GetIngress()
	if err != nil {
		return nil, fmt.Errorf("failed finding ingress for %s: %s", host, err)
	}
	app.deployment, err = app.GetDeployment()
	if err != nil {
		return nil, fmt.Errorf("failed finding deployment for %s: %s", host, err)
	}
	return app, nil
}

func (a *App) log(msg string) {
	a.logger.Printf("%s: %s", a.host, msg)
}

// GetIngress returns the ingress for the app
func (a *App) GetIngress() (*Ingress, error) {
	// Get all ingresses with an app label excluding the unidler ingress
	all, err := a.k8s.ExtensionsV1beta1().Ingresses("").List(metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name!=%s", UnidlerName),
		// TODO replace with
		// LabelSelector: fmt.Sprintf("host=%s", a.host),
		LabelSelector: "app",
	})
	if err != nil {
		return nil, fmt.Errorf("failed listing ingresses: %s", err)
	}

	// Search the list for an ingress with has a rule for the specified host.
	// TODO remove
	for _, ing := range all.Items {
		if ing.Spec.Rules[0].Host == a.host {
			a.log("Ingress found")
			ingress := Ingress(ing)
			return &ingress, nil
		}
	}

	return nil, fmt.Errorf("no ingress for host: %s", a.host)
}

// GetDeployment returns the deployment for the app
func (a *App) GetDeployment() (*Deployment, error) {
	// TODO replace with
	// deps, err := a.k8s.Apps().Deployments("").List(metav1.ListOptions{
	//     LabelSelector: fmt.Sprintf("host=%s", a.host),
	// })
	deps, err := a.k8s.Apps().Deployments(a.ingress.Namespace).List(
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("app=%s", a.ingress.Labels["app"]),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed listing deployments: %s", err)
	}
	num := len(deps.Items)
	if num != 1 {
		return nil, fmt.Errorf("want 1 deployment, got %d", num)
	}

	a.log("Deployment found")
	dep := Deployment(deps.Items[0])
	return &dep, nil
}

// SetReplicas updates an App's number of replicas to the specified number
func (a *App) SetReplicas() (err error) {
	// TODO change the annotation to number-of-replicas-when-not-idled and
	//      never remove it
	idledAt, exists := a.deployment.Annotations[IdledAtAnnotation]
	if !exists {
		// no annotation means the app is not idled, so skip this step
		return nil
	}

	// the idled-at annotation value is in the form <TIMESTAMP>,<NUM-REPLICAS>
	// TODO remove timestamp and just ParseInt
	num, err := strconv.ParseInt(strings.Split(idledAt, ",")[1], 10, 32)
	if err != nil {
		return fmt.Errorf("failed parsing idled-at annotation: %s", err)
	}

	replicas := int32(num)
	err = a.deployment.Patch(a.k8s, Replace("/spec/replicas", &replicas))
	if err != nil {
		return fmt.Errorf("failed setting replicas to %d: %s", replicas, err)
	}

	a.log(fmt.Sprintf("Deployment replicas changed to %d", replicas))
	return nil
}

// EnableIngress updates the App's ingress to route requests to the Deployment
func (a *App) EnableIngress(ingressClassName string) (err error) {
	if ingressClassName == "" {
		ingressClassName = "nginx"
	}

	err = a.ingress.Patch(
		a.k8s,
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
	ing, err := a.k8s.ExtensionsV1beta1().Ingresses(UnidlerNs).Get(
		UnidlerName,
		metav1.GetOptions{},
	)
	if err != nil {
		return fmt.Errorf("unidler ingress not found: %s", err)
	}
	ingress := Ingress(*ing)

	filteredRules := []v1beta1.IngressRule{}
	for _, rule := range ingress.Spec.Rules {
		if rule.Host != a.host {
			filteredRules = append(filteredRules, rule)
		}
	}

	err = ingress.Patch(
		a.k8s,
		Replace("/spec/rules", filteredRules),
	)
	if err != nil {
		return fmt.Errorf("failed updating unidler ingress rules: %s", err)
	}

	a.log("Removed from unidler ingress")
	return nil
}

// RemoveIdledMetadata removes the App's label and annotation which indicate its
// idled status, marking it as no longer idled
func (a *App) RemoveIdledMetadata() (err error) {
	// TODO change annotation to num-replicas-to-restore and never remove it
	err = a.deployment.Patch(
		a.k8s,
		Remove(JSONPointer("metadata", "annotations", IdledAtAnnotation)),
		Remove(JSONPointer("metadata", "labels", IdledLabel)),
	)
	if err != nil {
		// ignore missing label or annotation
		if !strings.Contains(err.Error(), "Unable to remove nonexistent key") {
			return fmt.Errorf("failed removing idled metadata: %s", err)
		}
	}

	a.log("Removed idled label/annotation")
	return nil
}

// WaitForDeployment blocks until the App's Deployment is ready to receive
// incoming requests
func (a *App) WaitForDeployment() error {
	w, err := a.deployment.Watch(a.k8s)
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
