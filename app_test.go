package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	coreAPI "k8s.io/api/core/v1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sFake "k8s.io/client-go/kubernetes/fake"
)

const (
	HOST = "test.example.com"
	NS   = "test-ns"
	NAME = "test"
)

var (
	app    *App
	deploy Deployment
	ing    Ingress
	svc    Service
)

func init() {
	k8sClient = k8sFake.NewSimpleClientset()

	// setup mock kubernetes resources
	deploy = mockDeployment(k8sClient, NS, NAME, HOST)
	svc = mockService(k8sClient, NS, NAME, HOST)
	ing = mockIngress(k8sClient, NS, NAME, HOST)

	app, _ = NewApp(HOST)
}

func TestNewApp(t *testing.T) {
	assert.Equal(t, &ing, app.ingress)
	assert.Equal(t, &deploy, app.deployment)
	assert.Equal(t, &svc, app.service)
}

func TestSetReplicas(t *testing.T) {
	// Check: We start with 0 replicas
	assert.Equal(t, int32(0), *deploy.Spec.Replicas)

	err := app.SetReplicas()

	assert.Nil(t, err)
	deploy = getDeployment(NS, NAME)
	assert.Equal(t, int32(1), *deploy.Spec.Replicas)
}

func TestRedirectService(t *testing.T) {
	// Check: We start in idled state (service points
	//        to unidler internal host)
	assert.Equal(t, coreAPI.ServiceTypeExternalName, svc.Spec.Type)
	assert.Equal(t, "unidler.default.svc.cluster.local", svc.Spec.ExternalName)

	err := app.RedirectService()

	assert.Nil(t, err)
	svc = getService(NS, NAME)
	assert.Equal(t, coreAPI.ServiceTypeClusterIP, svc.Spec.Type)
	// XXX fake patch doesn't remove
	//assert.Nil(t, svc.Spec.ExternalName)
	assert.Equal(t, NAME, svc.Spec.Selector["app"])
	assert.Equal(t, int32(80), svc.Spec.Ports[0].Port)
	assert.Equal(t, 3000, svc.Spec.Ports[0].TargetPort.IntValue())
}

func TestRemoveIdledMetadata(t *testing.T) {
	// Check: We have idled metadata
	assert.True(t, hasIdledLabel(deploy))
	assert.True(t, hasIdledAtAnnotation(deploy))

	err := app.RemoveIdledMetadata()

	assert.Nil(t, err)
	deploy = getDeployment(NS, NAME)
	// XXX fake patch doesn't remove
	// assert.False(t, hasIdledLabel(deploy))
	// assert.False(t, hasIdledAtAnnotation(deploy))
}

func hasIdledLabel(deploy Deployment) bool {
	_, ok := deploy.Labels[IdledLabel]
	return ok
}

func hasIdledAtAnnotation(dep Deployment) bool {
	_, ok := deploy.Annotations[IdledAtAnnotation]
	return ok
}

func getDeployment(ns string, name string) Deployment {
	dep, _ := k8sClient.AppsV1().Deployments(ns).Get(name, metaAPI.GetOptions{})
	return Deployment(*dep)
}

func getService(ns string, name string) Service {
	svc, _ := k8sClient.CoreV1().Services(ns).Get(name, metaAPI.GetOptions{})
	return Service(*svc)
}
