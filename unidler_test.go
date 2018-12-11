package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	testclient "k8s.io/client-go/kubernetes/fake"
)

type (
	MockSseSender struct {
		mock.Mock
	}
)

func (s *MockSseSender) SendSse(host string, msg *Message) {
	s.Called(host, msg)
}

func TestUnidleTask(t *testing.T) {
	ns := "test-ns"
	name := "test"
	host := "test.example.com"

	sse := new(MockSseSender)
	sse.On("SendSse", mock.Anything, mock.Anything).Return()

	client := testclient.NewSimpleClientset()
	k8s := &KubernetesAPI{
		client: client,
	}

	IdledDeployment(client, ns, name)
	MockIngress(client, ns, name, host)

	task := &UnidleTask{host: host, k8s: k8s, sse: sse}

	task.Fail(errors.New("test-error"))
	sse.AssertCalled(t, "SendSse", host, &Message{
		event: "error",
		data:  "test-error",
	})
}
