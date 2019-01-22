package main

import (
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

func GetIngress(k kubernetes.Interface, ns string, name string) Ingress {
	ing, _ := k.Extensions().Ingresses(ns).Get(name, metav1.GetOptions{})
	return Ingress(*ing)
}

func GetDeployment(k kubernetes.Interface, ns string, name string) Deployment {
	dep, _ := k.Apps().Deployments(ns).Get(name, metav1.GetOptions{})
	return Deployment(*dep)
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

	host := "test.example.com"
	ns := "test-ns"
	name := "test"

	// setup mock kubernetes resources
	// app deployment
	dep := Deployment(*IdledDeployment(client, ns, name))

	// app ingress
	ing := Ingress(*MockIngress(client, ns, name, host))

	app, _ := NewApp(host, client)
	assert.Equal(t, &ing, app.ingress)
	assert.Equal(t, &dep, app.deployment)

	assert.Equal(t, int32(0), *dep.Spec.Replicas)
	err := app.SetReplicas()
	assert.Nil(t, err)
	dep = GetDeployment(client, ns, name)
	assert.Equal(t, int32(1), *dep.Spec.Replicas)

	assert.Equal(t, "disabled", ing.Annotations[IngressClass])
	err = app.EnableIngress("nginx")
	assert.Nil(t, err)
	ing = GetIngress(client, ns, name)
	assert.Equal(t, "nginx", ing.Annotations[IngressClass])

	// setup unidler ingress
	ing = Ingress(*MockIngress(client, UnidlerNs, UnidlerName, host))
	assert.Equal(t, 1, countRulesForHost(ing, host))
	err = app.RemoveFromUnidlerIngress()
	assert.Nil(t, err)
	ing = GetIngress(client, UnidlerNs, UnidlerName)
	assert.Equal(t, 0, countRulesForHost(ing, host))
	assert.Equal(t, true, isIdled(dep))
	err = app.RemoveIdledMetadata()
	assert.Nil(t, err)
	dep = GetDeployment(client, ns, name)
	// XXX fake patch doesn't actually update metadata
	//assert.Equal(t, false, isIdled(dep))
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func countRulesForHost(ing Ingress, host string) int {
	count := 0
	for _, rule := range ing.Spec.Rules {
		if rule.Host == host {
			count++
		}
	}
	return count
}

func isIdled(dep Deployment) bool {
	_, labelExists := dep.Labels[IdledLabel]
	_, annotationExists := dep.Annotations[IdledAtAnnotation]
	return labelExists && annotationExists
}
