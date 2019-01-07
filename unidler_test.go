package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	testclient "k8s.io/client-go/kubernetes/fake"
)

type (
	MockSSESender struct {
		mock.Mock
	}
)

func (s *MockSSESender) SendSSE(msg *Message) {
	s.Called(msg)
}

func TestUnidler(t *testing.T) {
	ns := "test-ns"
	name := "test"
	host := "test.example.com"

	sse := new(MockSSESender)
	sse.On("SendSSE", mock.Anything).Return()

	client := testclient.NewSimpleClientset()
	k8s := &KubernetesAPI{
		client: client,
	}

	IdledDeployment(client, ns, name)
	MockIngress(client, ns, name, host)

	u := &Unidler{host: host, k8s: k8s, sse: sse}

	u.Fail(errors.New("test-error"))
	sse.AssertCalled(t, "SendSSE", &Message{
		event: "error",
		data:  "test-error",
	})
}
