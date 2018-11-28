package main

import (
	"fmt"
	"log"

	"k8s.io/api/apps/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type (
	// K8sClient is a wrapper client for the Kubernetes API
	K8sClient struct {
		client *kubernetes.Clientset
		config *rest.Config
	}

	// MockK8sClient is a mock of the K8sClient wrapper
	MockK8sClient struct{}

	// UnidlerK8sClient defines a minimal set of methods for a Kubernetes API
	// client for the Unidler
	UnidlerK8sClient interface {
		Deployment(*v1beta1.Ingress) (*v1.Deployment, error)
		Ingress(string, string) (*v1beta1.Ingress, error)
		IngressForHost(string) (*v1beta1.Ingress, error)
		ListIngresses(string, metav1.ListOptions) (*v1beta1.IngressList, error)
		ListDeployments(string, metav1.ListOptions) (*v1.DeploymentList, error)
		UpdateDeployment(*v1.Deployment) (*v1.Deployment, error)
		UpdateIngress(*v1beta1.Ingress) (*v1beta1.Ingress, error)
		WatchDeployment(*v1.Deployment) (watch.Interface, error)
	}
)

// NewK8sClient constructs a new K8sClient API wrapper
func NewK8sClient(path string) (k *K8sClient, err error) {
	k = &K8sClient{}
	if err = k.loadConfig(path); err != nil {
		return nil, err
	}
	if k.client, err = kubernetes.NewForConfig(k.config); err != nil {
		return nil, fmt.Errorf("Failed creating kubernetes client: %s", err)
	}
	return
}

func (k *K8sClient) loadConfig(path string) error {
	config, err := rest.InClusterConfig()
	if err == nil {
		k.config = config
		return nil
	}
	config, err = clientcmd.BuildConfigFromFlags("", path)
	if err == nil {
		k.config = config
		return nil
	}
	return fmt.Errorf("Failed loading kubernetes config: %s", err)
}

// ListIngresses returns a list of ingresses in the specified namespace with
// matching options (eg LabelSelector)
func (k K8sClient) ListIngresses(ns string, options metav1.ListOptions) (*v1beta1.IngressList, error) {
	return k.client.ExtensionsV1beta1().Ingresses(ns).List(options)
}

// ListDeployments returns a list of deployments in the specified namespace with
// matching options (eg LabelSelector)
func (k K8sClient) ListDeployments(ns string, options metav1.ListOptions) (*v1.DeploymentList, error) {
	return k.client.Apps().Deployments(ns).List(options)
}

// Ingress gets the Kubernetes ingress with the specified name in the specified
// namespace
func (k K8sClient) Ingress(ns string, name string) (*v1beta1.Ingress, error) {
	return k.client.ExtensionsV1beta1().Ingresses(ns).Get(name, metav1.GetOptions{})
}

// IngressForHost gets the Kubernetes ingress for the specified hostname
func (k K8sClient) IngressForHost(host string) (*v1beta1.Ingress, error) {
	// Get all ingresses excluding the unidler ingress
	ingresses, err := k.ListIngresses("", metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name!=%s", Unidler),
		LabelSelector: "app",
	})
	if err != nil {
		return nil, err
	}

	// NOTE: can't filter by spec.rules[0].host
	for _, ing := range ingresses.Items {
		if ing.Spec.Rules[0].Host == host {
			return &ing, nil
		}
	}

	return nil, fmt.Errorf("can't find ingress for host '%s'", host)
}

// Deployment gets the Kubernetes deployment for the specified ingress
func (k K8sClient) Deployment(ing *v1beta1.Ingress) (*v1.Deployment, error) {
	ns, app := ing.Namespace, ing.Labels["app"]
	deployments, err := k.ListDeployments(ns, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", app),
	})
	if err != nil {
		return nil, fmt.Errorf("can't retrieve list of deployments with label app='%s' (ns: '%s'): %s", app, ns, err)
	}

	if len(deployments.Items) != 1 {
		return nil, fmt.Errorf("expected exactly 1 deployment with label app='%s' (ns: '%s'), got %d instead", app, ns, len(deployments.Items))
	}

	dep := &deployments.Items[0]
	log.Printf("Deployment found '%s' (ns: '%s')\n", dep.Name, dep.Namespace)
	return dep, nil
}

// UpdateDeployment updates a deployment in kubernetes to match the specified
// Deployment
func (k K8sClient) UpdateDeployment(dep *v1.Deployment) (*v1.Deployment, error) {
	updated, err := k.client.Apps().Deployments(dep.Namespace).Update(dep)
	if err != nil {
		return nil, fmt.Errorf("failed updating deployment %s (ns: %s): %s", dep.Name, dep.Namespace, err)
	}
	return updated, nil
}

// UpdateIngress updates an ingress in kubernetes to match the specified Ingress
func (k K8sClient) UpdateIngress(ing *v1beta1.Ingress) (*v1beta1.Ingress, error) {
	updated, err := k.client.Extensions().Ingresses(ing.Namespace).Update(ing)
	if err != nil {
		return nil, fmt.Errorf("failed updating ingress %s (ns: %s): %s", ing.Name, ing.Namespace)
	}
	return updated, nil
}

// WatchDeployment gets a channel to watch a Deployment
func (k K8sClient) WatchDeployment(dep *v1.Deployment) (watch.Interface, error) {
	w, err := k.client.Apps().Deployments(dep.Namespace).Watch(metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name==%s", dep.Name),
	})
	if err != nil {
		return nil, fmt.Errorf("error while watching deployment '%s' (ns: '%s'): %s", dep.Name, dep.Namespace, err)
	}
	return w, nil
}
