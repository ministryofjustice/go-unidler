package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	appsAPI "k8s.io/api/apps/v1"
	coreAPI "k8s.io/api/core/v1"
	extAPI "k8s.io/api/extensions/v1beta1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"

	k8sFake "k8s.io/client-go/kubernetes/fake"
)

const (
	HOST = "test-tool.example.com"
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
	fmt.Println("DEBUG: testing.go init()...")

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
	assert.True(t, hasReplicasAnnotation(deploy))

	err := app.RemoveIdledMetadata()

	assert.Nil(t, err)
	deploy = getDeployment(NS, NAME)
	// XXX fake patch doesn't remove
	// assert.False(t, hasIdledLabel(deploy))
	// assert.False(t, hasReplicasAnnotation(deploy))
}

func hasIdledLabel(deploy Deployment) bool {
	_, ok := deploy.Labels[IdledLabel]
	return ok
}

func hasReplicasAnnotation(dep Deployment) bool {
	_, ok := deploy.Annotations[ReplicasWhenUnidledAnnotation]
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

func mockDeployment(client k8s.Interface, ns string, name string, host string) Deployment {
	var replicas int32
	deploy, _ := client.Apps().Deployments(ns).Create(&appsAPI.Deployment{
		ObjectMeta: metaAPI.ObjectMeta{
			Annotations: map[string]string{
				ReplicasWhenUnidledAnnotation: "1",
			},
			Name: name,
			Labels: map[string]string{
				"app":      name,
				"host":     host,
				IdledLabel: "true",
			},
		},
		Spec: appsAPI.DeploymentSpec{
			Replicas: &replicas,
		},
	})
	return Deployment(*deploy)
}

func mockIngress(client k8s.Interface, ns string, name string, host string) Ingress {
	ing, _ := client.Extensions().Ingresses(ns).Create(&extAPI.Ingress{
		ObjectMeta: metaAPI.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app":  name,
				"host": host,
			},
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "disabled",
			},
		},
		Spec: extAPI.IngressSpec{
			Rules: []extAPI.IngressRule{
				extAPI.IngressRule{
					Host: host,
				},
			},
		},
	})
	return Ingress(*ing)
}

func mockService(k k8s.Interface, ns string, name string, host string) Service {
	svc, _ := k.CoreV1().Services(ns).Create(&coreAPI.Service{
		ObjectMeta: metaAPI.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app":  name,
				"host": host,
			},
		},
		Spec: coreAPI.ServiceSpec{
			Type:         "ExternalName",
			ExternalName: "unidler.default.svc.cluster.local",
		},
	})
	return Service(*svc)
}
