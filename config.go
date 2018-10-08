package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NOTE: There is a nice package which uses tags to load configuration from
// environment variables but I didn't want to add a dependency just yet
//
// See:https://github.com/codingconcepts/env
type Config struct {
	Port             string `env:"PORT" default:":8080"`
	IngressClassName string `env:"INGRESS_CLASS_NAME" default:"nginx"`
	Logger           *log.Logger
	K8s              *kubernetes.Clientset
}

func LoadConfig() *Config {
	logger := NewLogger("unidler")

	k8sconfig, err := getK8sConfig()
	if err != nil {
		logger.Panicf("Couldn't get kubernetes config: %s", err)
	}

	k8s, err := kubernetes.NewForConfig(k8sconfig)
	if err != nil {
		logger.Panicf("Couldn't create kubernetes client: %s", err)
	}

	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = ":8080"
	}

	ingress_class_name, ok := os.LookupEnv("INGRESS_CLASS_NAME")
	if !ok {
		ingress_class_name = "nginx"
	}

	return &Config{
		Logger:           logger,
		K8s:              k8s,
		Port:             port,
		IngressClassName: ingress_class_name,
	}
}

func getK8sConfig() (*rest.Config, error) {
	k8sconfig, err := rest.InClusterConfig()
	if err == nil {
		return k8sconfig, nil
	}

	home := os.Getenv("HOME")
	if home == "" {
		return nil, fmt.Errorf("Couldn't determine HOME directory, is $HOME set?")
	}
	kubeconfig := filepath.Join(home, ".kube", "config")
	k8sconfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to get kubernetes config, both in-cluster and from ~/.kube/config: %s", err)
	}

	return k8sconfig, nil
}
