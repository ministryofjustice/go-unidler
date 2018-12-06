package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/api/apps/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
)

type (
	MockSseSender struct {
		mock.Mock
	}

	MockUnidler struct {
		mock.Mock
	}
)

func (s *MockSseSender) SendSse(host string, msg *Message) {
	s.Called(host, msg)
}

func (u *MockUnidler) EndTask(t *UnidleTask) {
	u.Called(t)
}

func (u *MockUnidler) SendSse(ch string, msg string) {
	u.Called(ch, msg)

}

func (u *MockUnidler) Unidle(host string) {
	u.Called(host)
}

func TestUnidleTask(t *testing.T) {
	ns := "test-ns"
	name := "test"
	host := "test.example.com"

	unidler := new(MockUnidler)
	unidler.On("EndTask", mock.Anything).Return()

	sse := new(MockSseSender)
	sse.On("SendSse", mock.Anything, mock.Anything).Return()

	client := testclient.NewSimpleClientset()
	api := &KubernetesAPI{
		client: client,
	}

	client.Apps().Deployments(ns).Create(&v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": name,
			},
		},
	})
	client.Extensions().Ingresses(ns).Create(&v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": name,
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

	app, err := NewApp(host, api)
	assert.Nil(t, err)
	task := &UnidleTask{app, host, sse, unidler}

	task.End()
	unidler.AssertCalled(t, "EndTask", task)

	task = &UnidleTask{app, host, sse, unidler}
	task.Fail(errors.New("test-error"))
	sse.AssertCalled(t, "SendSse", host, &Message{
		event: "error",
		data:  "test-error",
	})
	unidler.AssertCalled(t, "EndTask", task)
}

func TestUnidler(t *testing.T) {
	client := testclient.NewSimpleClientset()
	api := &KubernetesAPI{
		client: client,
	}
	sse := new(MockSseSender)
	unidler := NewUnidler("test-ns", "test", api, sse)
	assert.NotNil(t, unidler)
}
