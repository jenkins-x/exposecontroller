package main

import (
	"log"

	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
)

func main() {
	c, err := client.NewInCluster()
	if err != nil {
	 log.Fatalf("Cannot connect to api server: %v", err)
	}
	log.Printf("Connected")

	var fabric8 *api.Service
	fabric8, err = c.Services(api.NamespaceSystem).Get("fabric8")


	log.Printf("Connected")
	log.Printf(fabric8.Name)
}
