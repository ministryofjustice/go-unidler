package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	coreAPI "k8s.io/api/core/v1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	k8sFake "k8s.io/client-go/kubernetes/fake"
)

func getDeployment(k k8s.Interface, ns string, name string) Deployment {
	dep, _ := k.AppsV1().Deployments(ns).Get(name, metaAPI.GetOptions{})
	return Deployment(*dep)
}

func getService(k k8s.Interface, ns string, name string) Service {
	svc, _ := k.CoreV1().Services(ns).Get(name, metaAPI.GetOptions{})
	return Service(*svc)
}

func TestUnidleApp(t *testing.T) {
	client := k8sFake.NewSimpleClientset()

	host := "test.example.com"
	ns := "test-ns"
	name := "test"

	// setup mock kubernetes resources
	dep := mockDeployment(client, ns, name, host)
	ing := mockIngress(client, ns, name, host)
	svc := mockService(client, ns, name, host)

	app, _ := NewApp(host, client)
	assert.Equal(t, &ing, app.ingress)
	assert.Equal(t, &dep, app.deployment)

	assert.Equal(t, int32(0), *dep.Spec.Replicas)
	err := app.SetReplicas()
	assert.Nil(t, err)
	dep = getDeployment(client, ns, name)
	assert.Equal(t, int32(1), *dep.Spec.Replicas)

	assert.Equal(t, coreAPI.ServiceTypeExternalName, svc.Spec.Type)
	assert.Equal(t, "unidler.default.svc.cluster.local", svc.Spec.ExternalName)
	err = app.RedirectService()
	assert.Nil(t, err)
	svc = getService(client, ns, name)
	assert.Equal(t, coreAPI.ServiceTypeClusterIP, svc.Spec.Type)
	// XXX fake patch doesn't remove
	//assert.Nil(t, svc.Spec.ExternalName)
	assert.Equal(t, name, svc.Spec.Selector["app"])
	assert.Equal(t, int32(80), svc.Spec.Ports[0].Port)
	assert.Equal(t, 3000, svc.Spec.Ports[0].TargetPort.IntValue())

	assert.Equal(t, true, isIdled(dep))
	err = app.RemoveIdledMetadata()
	assert.Nil(t, err)
	dep = getDeployment(client, ns, name)
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
