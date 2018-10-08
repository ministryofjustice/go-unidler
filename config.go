package main

import (
	"log"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
	config := Config{
		Logger: NewLogger("unidler"),
	}

	k8sconfig, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	k8s, err := kubernetes.NewForConfig(k8sconfig)
	if err != nil {
		panic(err.Error())
	}

	config.K8s = k8s

	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = ":8080"
	}
	config.Port = port

	ingress_class_name, ok := os.LookupEnv("INGRESS_CLASS_NAME")
	if !ok {
		ingress_class_name = "nginx"
	}
	config.IngressClassName = ingress_class_name

	return &config
}
