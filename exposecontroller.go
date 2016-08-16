package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	rapi "github.com/openshift/origin/pkg/route/api"
	rapiv1 "github.com/openshift/origin/pkg/route/api/v1"
	"k8s.io/kubernetes/pkg/api"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

const (
	domain             = "domain"
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
		addExposerRule(c, svc, svc.Namespace)
	}
}

func serviceDeleted(obj interface{}) {
	svc, ok := obj.(cache.DeletedFinalStateUnknown)

	if ok {
		// service key is in the form namespace/name
		deleteExposerRule(svc.Key)
	} else {
		svc, ok := obj.(*api.Service)
		if ok {
			deleteExposerRule(svc.ObjectMeta.Name)
		} else {
			log.Fatalf("Error getting details of deleted service")
		}
	}
}

func addExposerRule(c *client.Client, svc *api.Service, ns string) {
	log.Println("Service created [" + svc.ObjectMeta.Name + "] in namespace [" + ns + "]")
	currentNs := os.Getenv("KUBERNETES_NAMESPACE")
	if len(currentNs) <= 0 {
		log.Fatalf("No KUBERNETES_NAMESPACE env var set")
	}

	environment, err := c.ConfigMaps(currentNs).Get(fabric8Environment)
	if err != nil {
		log.Fatalf("No ConfigMap with name [" + fabric8Environment + "] found in namespace [" + currentNs + "].  Was the exposer namespace setup by gofabric8?")
	}

	d, ok := environment.Data[domain]
	if !ok {
		log.Fatalf("No ConfigMap data with name [" + domain + "] found in namespace [" + currentNs + "].  Was the exposer namespace setup by gofabric8?")
	}

	switch environment.Data[exposerRule] {
	case ingress:
		err := createIngress(ns, d, svc, c)
		if err != nil {
			log.Fatalf("Unable to create ingress rule for service " + svc.ObjectMeta.Name)
		}
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

	ns := strings.Split(svc, "/")[0]
	name := strings.Split(svc, "/")[1]

	log.Println("Service deleted [" + name + "] in namespace [" + ns + "]")

}

func createIngress(ns string, domain string, service *api.Service, c *client.Client) error {
	rapi.AddToScheme(kapi.Scheme)
	rapiv1.AddToScheme(kapi.Scheme)

	ingressClient := c.Extensions().Ingress(ns)
	ingresses, err := ingressClient.List(kapi.ListOptions{})
	if err != nil {
		log.Printf("Failed to load ingresses in namespace %s with error %v", ns, err)
		return err
	}

	var labels = make(map[string]string)
	labels["provider"] = "fabric8"

	name := service.ObjectMeta.Name
	serviceSpec := service.Spec

	found := false

	// for now lets use the type of the service to know if we should create an ingress
	// TODO we should probably add an annotation to disable ingress creation
	if name != "jenkinshift" {
		for _, ingress := range ingresses.Items {
			if ingress.GetName() == name {
				found = true
				break
			}
			// TODO look for other ingresses with different names?
			for _, rule := range ingress.Spec.Rules {
				http := rule.HTTP
				if http != nil {
					for _, path := range http.Paths {
						ruleService := path.Backend.ServiceName
						if ruleService == name {
							found = true
							break
						}
					}
				}
			}
		}
		if !found {
			ports := serviceSpec.Ports
			hostName := name + "." + ns + "." + domain
			if len(ports) > 0 {
				rules := []extensions.IngressRule{}
				for _, port := range ports {
					rule := extensions.IngressRule{
						Host: hostName,
						IngressRuleValue: extensions.IngressRuleValue{
							HTTP: &extensions.HTTPIngressRuleValue{
								Paths: []extensions.HTTPIngressPath{
									{
										Backend: extensions.IngressBackend{
											ServiceName: name,
											// we need to use target port until https://github.com/nginxinc/kubernetes-ingress/issues/41 is fixed
											//ServicePort: intstr.FromInt(port.Port),
											ServicePort: port.TargetPort,
										},
									},
								},
							},
						},
					}
					rules = append(rules, rule)
				}
				ingress := extensions.Ingress{
					ObjectMeta: kapi.ObjectMeta{
						Labels: labels,
						Name:   name,
					},
					Spec: extensions.IngressSpec{
						Rules: rules,
					},
				}
				// lets create the ingress
				_, err = ingressClient.Create(&ingress)
				if err != nil {
					log.Printf("Failed to create the ingress %s with error %v", name, err)
					return err
				}
			}
		}
	}
	return nil
}
