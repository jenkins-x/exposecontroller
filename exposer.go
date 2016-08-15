package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

func main() {
	c, err := client.NewInCluster()
	if err != nil {
		log.Fatalf("Cannot connect to api server: %v", err)
	}
	log.Printf("Connected")
	_, controller := framework.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options api.ListOptions) (runtime.Object, error) {
				return c.Services(api.NamespaceAll).List(options)
			},
			WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
				return c.Services(api.NamespaceAll).Watch(options)
			},
		},
		&api.Service{},
		time.Millisecond*100,
		framework.ResourceEventHandlerFuncs{
			AddFunc:    serviceAdded,
			DeleteFunc: serviceDeleted,
		},
	)
	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(stop)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func serviceAdded(obj interface{}) {
	svc := obj.(*api.Service)
	addExposerRule(svc.ObjectMeta.Name)
}

func serviceDeleted(obj interface{}) {
	svc, ok := obj.(cache.DeletedFinalStateUnknown)
	if ok {
		// service key is in the form namespace/name
		deleteExposerRule(strings.Split(svc.Key, "/")[1])
	} else {
		svc, ok := obj.(*api.Service)
		if ok {
			deleteExposerRule(svc.ObjectMeta.Name)
		} else {
			log.Fatalf("Error getting details of deleted service")
		}
	}
}

func addExposerRule(svc string) {
	log.Println("Service created: " + svc)
}
func deleteExposerRule(svc string) {
	log.Println("Service deleted: " + svc)
}
