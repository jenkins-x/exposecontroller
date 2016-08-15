package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

const (
	exposerRule        = "exposer-rule"
	fabric8Environment = "fabric8-environment"
	ingress            = "ingress"
	loadBalancer       = "load-balancer"
	nodePort           = "node-port"
	route              = "route"
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
			AddFunc:    serviceAdded(c),
			DeleteFunc: serviceDeleted,
		},
	)
	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(stop)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func serviceAdded(c *client.Client) func(obj interface{}) {
	return func(obj interface{}) {
		svc := obj.(*api.Service)
		addExposerRule(c, svc.ObjectMeta.Name, svc.Namespace)
	}
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

func addExposerRule(c *client.Client, svc string, ns string) {
	log.Println("Service created [" + svc + "] in namespace [" + ns + "]")
	currentNs := os.Getenv("KUBERNETES_NAMESPACE")
	if len(currentNs) <= 0 {
		log.Fatalf("No KUBERNETES_NAMESPACE env var set")
	}

	environment, err := c.ConfigMaps(currentNs).Get(fabric8Environment)
	if err != nil {
		log.Fatalf("No ConfigMap with name [" + fabric8Environment + "] found in namespace [" + currentNs + "].  Was the exposer namespace setup by gofabric8?")
	}

	switch environment.Data[exposerRule] {
	case ingress:
		log.Println("Creating Ingress")
	case route:
		log.Println("Creating OpenShift Route")
	case nodePort:
		log.Println("Adapting Service type to be NodePort")
	case loadBalancer:
		log.Println("Adapting Service type to be LoadBalancer, this can take a few minutes to ve create by cloud provider")
	default:
		log.Fatalf("No match for [" + environment.Data[exposerRule] + "] exposer-rule found.  Was the exposer namespace setup by gofabric8?")
	}
}

func deleteExposerRule(svc string) {
	// TODO how do we get the namespace of the deleted service so we can find the corresponding rule / route to delete?
	log.Println("Service deleted [" + svc + "], not deleting the exposer rule yet until we figure out how to get the namespace from a DeletedFinalStateUnknown service object")
}
