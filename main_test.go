package main

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAllIngresses(t *testing.T) {
	k8s = mockK8sClientSet{}

	ingresses := allIngresses()
}

type (
	mockCall struct{}

	mockK8sIngressList struct {
		calls []mockCall
	}

	IngressLister interface {
		listIngresses(string, *metav1.ListOptions)
	}
)
