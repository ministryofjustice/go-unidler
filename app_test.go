package main

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func MockDeployment(client kubernetes.Interface, ns string, name string, annotations map[string]string, labels map[string]string) *v1.Deployment {
	var replicas int32
	labels["app"] = name
	dep, _ := client.Apps().Deployments(ns).Create(&v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
			Name:        name,
			Labels:      labels,
		},
		Spec: v1.DeploymentSpec{
			Replicas: &replicas,
		},
	})
	return dep
}

func MockIngress(client kubernetes.Interface, ns string, name string, host string) *v1beta1.Ingress {
	ing, _ := client.Extensions().Ingresses(ns).Create(&v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": name,
			},
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "disabled",
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				v1beta1.IngressRule{
					Host: host,
				},
			},
		},
	})
	return ing
}

func IdledDeployment(client kubernetes.Interface, ns string, name string) *v1.Deployment {
	annotations := map[string]string{
		IdledAtAnnotation: "2018-12-10T12:34:56Z,1",
	}
	labels := map[string]string{
		IdledLabel: "true",
	}
	return MockDeployment(client, ns, name, annotations, labels)
}

func TestUnidleApp(t *testing.T) {
	client := testclient.NewSimpleClientset()
	api := &KubernetesAPI{
		client: client,
		log:    log.New(os.Stdout, "", log.Lshortfile),
	}

	host := "test.example.com"
	ns := "test-ns"
	name := "test"

	// setup mock kubernetes resources
	// app deployment
	dep := IdledDeployment(client, ns, name)

	// app ingress
	ing := MockIngress(client, ns, name, host)

	app, _ := NewApp(host, api)
	assert.Equal(t, ing, app.ingress)
	assert.Equal(t, dep, app.deployment)

	var expectedReplicas int32 = 1
	err := app.SetReplicas()
	assert.Nil(t, err)
	assert.Equal(t, expectedReplicas, *app.deployment.Spec.Replicas)

	err = app.EnableIngress("nginx")
	assert.Nil(t, err)
	assert.Equal(t, "nginx", app.ingress.Annotations["kubernetes.io/ingress.class"])

	// setup unidler ingress
	MockIngress(client, UnidlerNs, UnidlerName, host)
	err = app.RemoveFromUnidlerIngress()
	assert.Nil(t, err)
	assert.Equal(t, false, ingressRuleExists(host, api))

	err = app.RemoveIdledMetadata()
	assert.Nil(t, err)
	// XXX fake does not perform remove operation :(
	//_, labelExists := app.deployment.Labels[IdledLabel]
	//_, annotationExists := app.deployment.Annotations[IdledAtAnnotation]
	//assert.False(t, labelExists, "Idled label not removed")
	//assert.False(t, annotationExists, "Idled annotation not removed")
	assert.Equal(t, expectedReplicas, *app.deployment.Spec.Replicas)
}

func ingressRuleExists(host string, api *KubernetesAPI) bool {
	ing, _ := api.Ingress(UnidlerNs, UnidlerName)
	_, found := removeHostRule(host, ing.Spec.Rules)
	return found
}
