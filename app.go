package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	appsAPI "k8s.io/api/apps/v1"
	coreAPI "k8s.io/api/core/v1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/ministryofjustice/analytics-platform-go-unidler/jsonpatch"
)

// App is a Analytical Platform "app" consisting of a kubernetes
// deployment, with a corresponding hostname and ingress
type App struct {
	deployment *Deployment
	host       string
	ingress    *Ingress
	logger     *log.Logger
	service    *Service
}

const (
	// IdledLabel is a metadata label which indicates a Deployment is idled.
	IdledLabel = "mojanalytics.xyz/idled"
	// IdledAtAnnotation is a metadata annotation which indicates the time a
	// Deployment was idled and the number of replicas it had at that time,
	// separated by a semicolon, eg: "2018-11-26T17:27:34;2".
	IdledAtAnnotation = "mojanalytics.xyz/idled-at"
	// UnidlerName is the name of the kubernetes Unidler ingress
	UnidlerName = "unidler"
	// UnidlerNs is the namespace of the kubernetes Unidler ingress
	UnidlerNs = "default"
)

// NewApp constructs a new App and fetches the corresponding kubernetes ingress
// and deployment
func NewApp(host string) (app *App, err error) {
	app = &App{
		host:   host,
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
	app.service, err = app.GetService()
	if err != nil {
		return nil, fmt.Errorf("failed finding service for %s: %s", host, err)
	}
	return app, nil
}

func (a *App) log(msg string) {
	a.logger.Printf("%s: %s", a.host, msg)
}

// GetIngress returns the ingress for the app
func (a *App) GetIngress() (*Ingress, error) {
	// Get all ingresses with an app label excluding the unidler ingress
	all, err := k8sClient.ExtensionsV1beta1().Ingresses("").List(metaAPI.ListOptions{
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
	// deps, err := k8sClient.Apps().Deployments("").List(metaAPI.ListOptions{
	//     LabelSelector: fmt.Sprintf("host=%s", a.host),
	// })
	deps, err := k8sClient.AppsV1().Deployments(a.ingress.Namespace).List(
		metaAPI.ListOptions{
			LabelSelector: fmt.Sprintf("app=%s", a.ingress.Labels["app"]),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed listing deployments: %s", err)
	}
	num := len(deps.Items)
	if num != 1 {
		return nil, fmt.Errorf("expected exactly 1 Deployment, found %d", num)
	}

	a.log("Deployment found")
	dep := Deployment(deps.Items[0])
	return &dep, nil
}

// GetService returns the service for the app
func (a *App) GetService() (*Service, error) {
	// TODO replace with
	// svcs, err := k8sClient.CoreV1().Services("").List(metaAPI.ListOptions{
	//     LabelSelector: fmt.Sprintf("host=%s", a.host),
	// })
	svcs, err := k8sClient.CoreV1().Services(a.ingress.Namespace).List(
		metaAPI.ListOptions{
			LabelSelector: fmt.Sprintf("app=%s", a.ingress.Labels["app"]),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed listing deployments: %s", err)
	}

	a.log("Service found")
	svc := Service(svcs.Items[0])
	return &svc, nil
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
	patch := jsonpatch.Patch(
		jsonpatch.Replace([]string{"spec", "replicas"}, &replicas),
	)
	err = a.deployment.Patch(patch)
	if err != nil {
		return fmt.Errorf("failed setting replicas to %d: %s", replicas, err)
	}

	a.log(fmt.Sprintf("Deployment replicas changed to %d", replicas))
	return nil
}

// RedirectService redirects the App's service from the unidler to the app pods
func (a *App) RedirectService() error {
	patch := jsonpatch.Patch(
		jsonpatch.Remove([]string{"spec", "externalName"}),
		jsonpatch.Replace([]string{"spec", "type"}, string(coreAPI.ServiceTypeClusterIP)),
		jsonpatch.Add([]string{"spec", "selector"}, &map[string]string{
			"app": a.service.Labels["app"],
		}),
		jsonpatch.Add([]string{"spec", "ports"}, []coreAPI.ServicePort{
			coreAPI.ServicePort{
				Port:       int32(80),
				TargetPort: intstr.FromInt(3000),
			},
		}),
	)

	err := a.service.Patch(patch)
	if err != nil {
		return fmt.Errorf("failed redirecting service: %s", err)
	}

	a.log("Service redirected")
	return nil
}

// RemoveIdledMetadata removes the App's label and annotation which indicate its
// idled status, marking it as no longer idled
func (a *App) RemoveIdledMetadata() (err error) {
	patch := jsonpatch.Patch(
		jsonpatch.Remove([]string{"metadata", "annotations", IdledAtAnnotation}),
		jsonpatch.Remove([]string{"metadata", "labels", IdledLabel}),
	)

	// TODO change annotation to num-replicas-to-restore and never remove it
	err = a.deployment.Patch(patch)
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
	w, err := a.deployment.Watch()
	if err != nil {
		return fmt.Errorf("failed watching deployment: %s", err)
	}

	for event := range w.ResultChan() {
		dep, ok := event.Object.(*appsAPI.Deployment)
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
