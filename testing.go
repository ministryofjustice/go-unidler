package main

import (
	appsAPI "k8s.io/api/apps/v1"
	coreAPI "k8s.io/api/core/v1"
	extAPI "k8s.io/api/extensions/v1beta1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
)

func mockDeployment(client k8s.Interface, ns string, name string, host string) Deployment {
	var replicas int32
	dep, _ := client.Apps().Deployments(ns).Create(&appsAPI.Deployment{
		ObjectMeta: metaAPI.ObjectMeta{
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
		Spec: appsAPI.DeploymentSpec{
			Replicas: &replicas,
		},
	})
	return Deployment(*dep)
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
