package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func MockDeployment(client kubernetes.Interface, ns string, name string, host string) Deployment {
	var replicas int32
	dep, _ := client.Apps().Deployments(ns).Create(&v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				IdledAtAnnotation: "2018-12-10T12:34:56Z,1",
			},
			Name: name,
			Labels: map[string]string{
				"app":      name,
				"host":     host,
				IdledLabel: "true",
			},
		},
		Spec: v1.DeploymentSpec{
			Replicas: &replicas,
		},
	})
	return Deployment(*dep)
}

func MockIngress(client kubernetes.Interface, ns string, name string, host string) Ingress {
	ing, _ := client.Extensions().Ingresses(ns).Create(&v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app":  name,
				"host": host,
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
	return Ingress(*ing)
}

func MockService(k kubernetes.Interface, ns string, name string, host string) Service {
	svc, _ := k.CoreV1().Services(ns).Create(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app":  name,
				"host": host,
			},
		},
		Spec: corev1.ServiceSpec{
			Type:         "ExternalName",
			ExternalName: "unidler.default.svc.cluster.local",
		},
	})
	return Service(*svc)
}

func GetIngress(k kubernetes.Interface, ns string, name string) Ingress {
	ing, _ := k.ExtensionsV1beta1().Ingresses(ns).Get(name, metav1.GetOptions{})
	return Ingress(*ing)
}

func GetDeployment(k kubernetes.Interface, ns string, name string) Deployment {
	dep, _ := k.AppsV1().Deployments(ns).Get(name, metav1.GetOptions{})
	return Deployment(*dep)
}

func GetService(k kubernetes.Interface, ns string, name string) Service {
	svc, _ := k.CoreV1().Services(ns).Get(name, metav1.GetOptions{})
	return Service(*svc)
}

func TestUnidleApp(t *testing.T) {
	client := testclient.NewSimpleClientset()

	host := "test.example.com"
	ns := "test-ns"
	name := "test"

	// setup mock kubernetes resources
	dep := MockDeployment(client, ns, name, host)
	ing := MockIngress(client, ns, name, host)
	svc := MockService(client, ns, name, host)

	app, _ := NewApp(host, client)
	assert.Equal(t, &ing, app.ingress)
	assert.Equal(t, &dep, app.deployment)

	assert.Equal(t, int32(0), *dep.Spec.Replicas)
	err := app.SetReplicas()
	assert.Nil(t, err)
	dep = GetDeployment(client, ns, name)
	assert.Equal(t, int32(1), *dep.Spec.Replicas)

	assert.Equal(t, corev1.ServiceTypeExternalName, svc.Spec.Type)
	assert.Equal(t, "unidler.default.svc.cluster.local", svc.Spec.ExternalName)
	err = app.RedirectService()
	assert.Nil(t, err)
	svc = GetService(client, ns, name)
	assert.Equal(t, corev1.ServiceTypeClusterIP, svc.Spec.Type)
	// XXX fake patch doesn't remove
	//assert.Nil(t, svc.Spec.ExternalName)
	assert.Equal(t, name, svc.Spec.Selector["app"])
	assert.Equal(t, int32(80), svc.Spec.Ports[0].Port)
	assert.Equal(t, 3000, svc.Spec.Ports[0].TargetPort.IntValue())

	assert.Equal(t, true, isIdled(dep))
	err = app.RemoveIdledMetadata()
	assert.Nil(t, err)
	dep = GetDeployment(client, ns, name)
	// XXX fake patch doesn't remove
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
