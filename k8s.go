package main

import (
	"fmt"

	appsAPI "k8s.io/api/apps/v1"
	coreAPI "k8s.io/api/core/v1"
	extAPI "k8s.io/api/extensions/v1beta1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	k8s "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type (
	// Deployment wraps appsAPI.Deployment to add methods
	Deployment appsAPI.Deployment

	// Ingress wraps extAPI.Ingress to add methods
	Ingress extAPI.Ingress

	// Service wraps coreAPI.Service to add methods
	Service coreAPI.Service
)

// KubernetesClient constructs a new Kubernetes client
func KubernetesClient(path string) (k k8s.Interface, err error) {
	config, err := loadConfig(path)
	if err != nil {
		return nil, fmt.Errorf("failed creating kubernetes client: %s", err)
	}
	client, err := k8s.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed creating kubernetes client: %s", err)
	}
	return client, nil
}

func loadConfig(path string) (config *rest.Config, err error) {
	config, err = rest.InClusterConfig()
	if err == nil {
		return
	}
	config, err = clientcmd.BuildConfigFromFlags("", path)
	if err == nil {
		return
	}
	return nil, fmt.Errorf("failed loading kubernetes config: %s", err)
}

// Patch applies a JSON patch to a Deployment
func (d *Deployment) Patch(patch []byte) error {
	_, err := k8sClient.Apps().Deployments(d.Namespace).Patch(
		d.Name,
		types.JSONPatchType,
		patch,
	)
	if err != nil {
		return fmt.Errorf("failed to patch Deployment: %s", err)
	}

	return nil
}

// Patch applies a JSON patch to a Service
func (svc *Service) Patch(patch []byte) error {
	_, err := k8sClient.CoreV1().Services(svc.Namespace).Patch(
		svc.Name,
		types.JSONPatchType,
		patch,
	)
	if err != nil {
		return fmt.Errorf("failed to patch Service: %s", err)
	}
	return nil
}

// Watch gets a channel to watch a Deployment
func (dep *Deployment) Watch() (watch.Interface, error) {
	return k8sClient.Apps().Deployments(dep.Namespace).Watch(metaAPI.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name==%s", dep.Name),
	})
}
