package main

import (
	"fmt"
	"log"
	"os"

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
	// KubernetesAPI is a wrapper client for the Kubernetes API
	KubernetesAPI struct {
		client kubernetes.Interface
		config *rest.Config
		log    *log.Logger
	}
)

// NewKubernetesAPI constructs a new KubernetesAPI
func NewKubernetesAPI(path string) (k *KubernetesAPI, err error) {
	k = &KubernetesAPI{}
	if err = k.loadConfig(path); err != nil {
		return nil, fmt.Errorf("failed creating kubernetes client: %s", err)
	}
	if k.client, err = kubernetes.NewForConfig(k.config); err != nil {
		return nil, fmt.Errorf("failed creating kubernetes client: %s", err)
	}
	k.log = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	return
}

func (k *KubernetesAPI) loadConfig(path string) error {
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
	return fmt.Errorf("failed loading kubernetes config: %s", err)
}

// ListIngresses returns a list of ingresses in the specified namespace with
// matching options (eg LabelSelector)
func (k KubernetesAPI) ListIngresses(ns string, options metav1.ListOptions) (*v1beta1.IngressList, error) {
	return k.client.ExtensionsV1beta1().Ingresses(ns).List(options)
}

// ListDeployments returns a list of deployments in the specified namespace with
// matching options (eg LabelSelector)
func (k KubernetesAPI) ListDeployments(ns string, options metav1.ListOptions) (*v1.DeploymentList, error) {
	return k.client.Apps().Deployments(ns).List(options)
}

// Ingress gets the Kubernetes ingress with the specified name in the specified
// namespace
func (k KubernetesAPI) Ingress(ns string, name string) (*v1beta1.Ingress, error) {
	return k.client.ExtensionsV1beta1().Ingresses(ns).Get(name, metav1.GetOptions{})
}

// IngressForHost gets the Kubernetes ingress for the specified hostname
func (k KubernetesAPI) IngressForHost(host string) (*v1beta1.Ingress, error) {
	// Get all ingresses with an app label excluding the unidler ingress
	ingresses, err := k.ListIngresses("", metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name!=%s", UnidlerName),
		LabelSelector: "app",
	})
	if err != nil {
		return nil, err
	}

	// Search the list for an ingress which has a rule for the specified host.
	// Unfortunatly, kubernetes field selectors do not work for this.
	// TODO: add a host label to app ingresses would allow us to ListIngresses
	// where LabelSelector "host={host}"
	for _, ing := range ingresses.Items {
		// XXX assumes the ingress has only one rule
		if ing.Spec.Rules[0].Host == host {
			k.log.Printf("Ingress for '%s' found: %s (ns: %s)", host, ing.Name, ing.Namespace)
			return &ing, nil
		}
	}

	return nil, fmt.Errorf("can't find ingress for host '%s'", host)
}

// Deployment gets the Kubernetes deployment for the specified ingress
func (k KubernetesAPI) Deployment(ing *v1beta1.Ingress) (*v1.Deployment, error) {
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
	k.log.Printf("Deployment found '%s' (ns: '%s')\n", dep.Name, dep.Namespace)
	return dep, nil
}

// PatchDeployment patches a deployment in kubernetes with a specified JSON
// patch
func (k KubernetesAPI) PatchDeployment(dep *v1.Deployment, patch string) (*v1.Deployment, error) {
	return k.client.Apps().Deployments(dep.Namespace).Patch(dep.Name, types.JSONPatchType, []byte(patch))
}

// UpdateDeployment updates a deployment in kubernetes to match the specified
// Deployment
func (k KubernetesAPI) UpdateDeployment(dep *v1.Deployment) (*v1.Deployment, error) {
	return k.client.Apps().Deployments(dep.Namespace).Update(dep)
}

// UpdateIngress updates an ingress in kubernetes to match the specified Ingress
func (k KubernetesAPI) UpdateIngress(ing *v1beta1.Ingress) (*v1beta1.Ingress, error) {
	return k.client.Extensions().Ingresses(ing.Namespace).Update(ing)
}

// WatchDeployment gets a channel to watch a Deployment
func (k KubernetesAPI) WatchDeployment(dep *v1.Deployment) (watch.Interface, error) {
	return k.client.Apps().Deployments(dep.Namespace).Watch(metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name==%s", dep.Name),
	})
}
