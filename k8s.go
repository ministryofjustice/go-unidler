package main

import (
	"encoding/json"
	"fmt"

	"k8s.io/api/apps/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type (
	// Deployment wraps v1.Deployment to add methods
	Deployment v1.Deployment

	// Ingress wraps v1beta1.Ingress to add methods
	Ingress v1beta1.Ingress
)

// KubernetesClient constructs a new Kubernetes client
func KubernetesClient(path string) (k *kubernetes.Clientset, err error) {
	config, err := loadConfig(path)
	if err != nil {
		return nil, fmt.Errorf("failed creating kubernetes client: %s", err)
	}
	client, err := kubernetes.NewForConfig(config)
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
func (dep *Deployment) Patch(k kubernetes.Interface, p ...*Operation) error {
	bytes, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed parsing patch: %s", err)
	}

	_, err = k.Apps().Deployments(dep.Namespace).Patch(
		dep.Name,
		types.JSONPatchType,
		bytes,
	)
	if err != nil {
		return fmt.Errorf("failed patching deployment: %s", err)
	}
	return nil
}

// Patch applies a JSONPatch to an Ingress
func (ing *Ingress) Patch(k kubernetes.Interface, p ...*Operation) error {
	bytes, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed parsing patch: %s", err)
	}
	_, err = k.Extensions().Ingresses(ing.Namespace).Patch(
		ing.Name,
		types.JSONPatchType,
		bytes,
	)
	if err != nil {
		return fmt.Errorf("failed patching ingress: %s", err)
	}
	return nil
}

// Watch gets a channel to watch a Deployment
func (dep *Deployment) Watch(k kubernetes.Interface) (watch.Interface, error) {
	return k.Apps().Deployments(dep.Namespace).Watch(metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name==%s", dep.Name),
	})
}
