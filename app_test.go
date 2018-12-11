package main

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func TestJsonPatchEscape(t *testing.T) {
	assert := assert.New(t)
	cases := []struct {
		value    string
		expected string
	}{
		{"foo", "foo"},
		{"foo/bar", "foo~1bar"},
		{"foo/bar~1", "foo~1bar~01"},
		{"foo/bar/quux/baz", "foo~1bar~1quux~1baz"},
		{"/////", "~1~1~1~1~1"},
		{"~~~~~", "~0~0~0~0~0"},
		{"", ""},
	}

	for _, c := range cases {
		assert.Equal(jsonPatchEscape(c.value), c.expected)
	}
}

func IdledDeployment(client kubernetes.Interface, ns string, name string, annotations map[string]string, labels map[string]string) *v1.Deployment {
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

func TestUnidleApp(t *testing.T) {
	client := testclient.NewSimpleClientset()
	api := &KubernetesAPI{
		client: client,
	}

	host := "test.example.com"
	ns := "test-ns"
	name := "test"
	var expectedReplicas int32 = 1

	// setup mock kubernetes resources
	// app deployment
	annotations := map[string]string{
		IdledAtAnnotation: "2018-12-10T12:34:56Z,1",
	}
	labels := map[string]string{
		IdledLabel: "true",
	}
	dep := IdledDeployment(client, ns, name, annotations, labels)
	// app ingress
	ing := MockIngress(client, ns, name, host)

	app, _ := NewApp(host, api)
	assert.Equal(t, ing, app.ingress)
	assert.Equal(t, dep, app.deployment)

	err := app.SetReplicas(1)
	assert.Nil(t, err)
	assert.Equal(t, expectedReplicas, *app.deployment.Spec.Replicas)

	err = app.EnableIngress("nginx")
	assert.Nil(t, err)
	assert.Equal(t, "nginx", app.ingress.Annotations["kubernetes.io/ingress.class"])

	// unidler ingress
	MockIngress(client, UnidlerNs, UnidlerName, host)

	err = app.RemoveFromUnidlerIngress()
	assert.Nil(t, err)
	unidlerIngress, _ := client.ExtensionsV1beta1().Ingresses(UnidlerNs).Get(UnidlerName, metav1.GetOptions{})
	count := 0
	for _, r := range unidlerIngress.Spec.Rules {
		if r.Host == host {
			count++
		}
	}
	assert.Equal(t, 0, count)

	err = app.RemoveIdledMetadata()
	assert.Nil(t, err)

	dep, _ = client.Apps().Deployments(ns).Get(name, metav1.GetOptions{})
	l, labelExists := dep.Labels[IdledLabel]
	log.Print(l)
	a, annotationExists := dep.Annotations[IdledAtAnnotation]
	log.Print(a)
	assert.False(t, labelExists, "Idled label not removed")
	assert.False(t, annotationExists, "Idled annotation not removed")
}
