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

	jp "github.com/ministryofjustice/analytics-platform-go-unidler/jsonpatch"
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
	// ReplicasWhenUnidledAnnotation contains the number of replicas an app
	// had before being idled
	ReplicasWhenUnidledAnnotation = "mojanalytics.xyz/replicas-when-unidled"
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
		app.log("Ingress not found: %s", err)
		return nil, fmt.Errorf("Ingress for your app not found.")
	}
	app.deployment, err = app.GetDeployment()
	if err != nil {
		app.log("Deployment not found: %s", err)
		return nil, fmt.Errorf("Deployment for your app not found.")
	}
	app.service, err = app.GetService()
	if err != nil {
		app.log("Service not found: %s", err)
		return nil, fmt.Errorf("Service for your app not found.")
	}
	return app, nil
}

func (a *App) log(format string, args ...interface{}) {
	a.logger.Printf("%s: %s", a.host, fmt.Sprintf(format, args...))
}

// GetIngress returns the ingress for the app
func (a *App) GetIngress() (*Ingress, error) {
	// Get ingresses with app host label
	ings, err := k8sClient.ExtensionsV1beta1().Ingresses("").List(metaAPI.ListOptions{
		LabelSelector: fmt.Sprintf("host=%s", a.host),
	})
	if err != nil {
		return nil, fmt.Errorf("failed listing ingresses: %s", err)
	}

	count := len(ings.Items)
	if count != 1 {
		return nil, fmt.Errorf("expected exactly 1 Ingress with host label, found %d", count)
	}

	a.log("Ingress found.")
	ingress := Ingress(ings.Items[0])
	return &ingress, nil
}

// GetDeployment returns the deployment for the app
func (a *App) GetDeployment() (*Deployment, error) {
	deps, err := k8sClient.AppsV1().Deployments(a.ingress.Namespace).List(
		metaAPI.ListOptions{
			LabelSelector: fmt.Sprintf("host=%s", a.host),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed listing deployments: %s", err)
	}
	count := len(deps.Items)
	if count != 1 {
		return nil, fmt.Errorf("expected exactly 1 Deployment with host label, found %d", count)
	}

	a.log("Deployment found.")
	dep := Deployment(deps.Items[0])
	return &dep, nil
}

// GetService returns the service for the app
func (a *App) GetService() (*Service, error) {
	svcs, err := k8sClient.CoreV1().Services(a.ingress.Namespace).List(
		metaAPI.ListOptions{
			LabelSelector: fmt.Sprintf("host=%s", a.host),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed listing services: %s", err)
	}

	count := len(svcs.Items)
	if count != 1 {
		return nil, fmt.Errorf("expected exactly 1 Service with host label, found %d", count)
	}

	a.log("Service found.")
	svc := Service(svcs.Items[0])
	return &svc, nil
}

// GetReplicasWhenUnidled return the number of replicas when app is unidled
func (a *App) GetReplicasWhenUnidled() (replicas int) {
	replicasWhenUnidled, exists := a.deployment.Annotations[ReplicasWhenUnidledAnnotation]
	if !exists {
		a.log("Deployment doesn't have '%s' annotation. Assuming it had 1 replica when it was unidled.", ReplicasWhenUnidledAnnotation)
		return 1
	}

	num, err := strconv.ParseInt(replicasWhenUnidled, 10, 32)
	if err != nil {
		a.log("Failed to parse number of replicas when unidled, assuming Deployment had 1 replica. Deployment annotation: '%s=%s': %s", ReplicasWhenUnidledAnnotation, replicasWhenUnidled, err)
		return 1
	}

	if num < 1 {
		a.log("Replicas when unidled was %d, assuming Deployment had 1 replica: '%s=%s'.", num, ReplicasWhenUnidledAnnotation, replicasWhenUnidled)
		return 1
	}

	return int(num)
}

// SetReplicas updates an App's number of replicas to the specified number
func (a *App) SetReplicas() (err error) {
	if *a.deployment.Spec.Replicas > 0 {
		a.log("Deployment's replicas is already %d. Assuming is already unidled.", *a.deployment.Spec.Replicas)
		return nil
	}

	num := a.GetReplicasWhenUnidled()

	replicas := int32(num)
	patch := jp.Patch(
		jp.Replace(jp.Path("spec", "replicas"), &replicas),
	)
	err = a.deployment.Patch(patch)
	if err != nil {
		a.log("Patch to set replicas back to %d failed: %s", replicas, err)
		return fmt.Errorf("Failed to set your app's replicas back to %d.", replicas)
	}

	a.log("Successfully set Deployment's replicas to %d.", replicas)
	return nil
}

// RedirectService redirects the App's service from the unidler to the app pods
func (a *App) RedirectService() error {
	patch := jp.Patch(
		jp.Remove(jp.Path("spec", "externalName")),
		jp.Replace(jp.Path("spec", "type"), string(coreAPI.ServiceTypeClusterIP)),
		jp.Add(jp.Path("spec", "selector"), &map[string]string{
			"app": a.service.Labels["app"],
		}),
		jp.Add(jp.Path("spec", "ports"), []coreAPI.ServicePort{
			coreAPI.ServicePort{
				Port:       int32(80),
				TargetPort: intstr.FromInt(3000),
			},
		}),
	)

	err := a.service.Patch(patch)
	if err != nil {
		a.log("Patch to Service failed: %s", err)

		// ignore missing label or annotation
		if strings.Contains(err.Error(), "Unable to remove nonexistent key") {
			a.log("Ignored Service Patch error caused by nonexistent key.")
			return nil
		}

		return fmt.Errorf("Failed to redirect back your app.")
	}

	a.log("Successfully redirected Service back to app's pods.")
	return nil
}

// RemoveIdledMetadata removes the App's label and annotation which indicate its
// idled status, marking it as no longer idled
func (a *App) RemoveIdledMetadata() (err error) {
	patch := jp.Patch(
		jp.Remove(jp.Path("metadata", "annotations", IdledAtAnnotation)),
		jp.Remove(jp.Path("metadata", "annotations", ReplicasWhenUnidledAnnotation)),
		jp.Remove(jp.Path("metadata", "labels", IdledLabel)),
	)

	// TODO change annotation to num-replicas-to-restore and never remove it
	err = a.deployment.Patch(patch)
	if err != nil {
		a.log("Patch to remove idled metadata label/annotation failed: %s", err)

		// ignore missing label or annotation
		if strings.Contains(err.Error(), "Unable to remove nonexistent key") {
			a.log("Ignored Deployment Patch error caused by nonexistent key.")
			return nil
		}

		return fmt.Errorf("Failed to remove idled metadata from your app.")
	}

	a.log("Successfully removed idled metadata (label/annotation) from Deployment.")
	return nil
}

// WaitForDeployment blocks until the App's Deployment is ready to receive
// incoming requests
func (a *App) WaitForDeployment() error {
	userFriendlyError := fmt.Errorf("Failed to wait for for your app to come back up.")

	w, err := a.deployment.Watch()
	if err != nil {
		a.log("Watch on Deployment failed: %s", err)
		return userFriendlyError
	}

	for event := range w.ResultChan() {
		dep, ok := event.Object.(*appsAPI.Deployment)
		if !ok {
			a.log("Unexpected Watch event type: %+v", event.Object)
			return userFriendlyError
		}

		if dep.Status.AvailableReplicas > 0 {
			break
		}
	}

	a.log("Successfully waited for Deployment replicas to be available.")
	return nil
}
