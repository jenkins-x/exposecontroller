package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/daviddengcn/go-colortext"
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
	exposeRule         = "expose-rule"
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
	success("Connected")
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
			DeleteFunc: serviceDeleted(c),
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
		addExposeRule(c, svc, svc.Namespace)
	}
}

func serviceDeleted(c *client.Client) func(obj interface{}) {
	return func(obj interface{}) {
		svc, ok := obj.(cache.DeletedFinalStateUnknown)

		if ok {
			// service key is in the form namespace/name
			deleteExposeRule(svc.Key, c)
		} else {
			svc, ok := obj.(*api.Service)
			if ok {
				deleteExposeRule(svc.ObjectMeta.Name, c)
			} else {
				log.Fatalf("Error getting details of deleted service")
			}
		}
	}
}

func addExposeRule(c *client.Client, svc *api.Service, ns string) {
	log.Println("Found service [" + svc.ObjectMeta.Name + "] in namespace [" + ns + "]")
	currentNs := os.Getenv("KUBERNETES_NAMESPACE")
	if len(currentNs) <= 0 {
		log.Fatalf("No KUBERNETES_NAMESPACE env var set")
	}

	environment, err := c.ConfigMaps(currentNs).Get(fabric8Environment)
	if err != nil {
		log.Fatalf("No ConfigMap with name [" + fabric8Environment + "] found in namespace [" + currentNs + "].  Was the exposecontroller namespace setup by gofabric8?")
	}

	d, ok := environment.Data[domain]
	if !ok {
		log.Fatalf("No ConfigMap data with name [" + domain + "] found in namespace [" + currentNs + "].  Was the exposecontroller namespace setup by gofabric8?")
	}

	switch environment.Data[exposeRule] {
	case ingress:
		err := createIngress(ns, d, svc, c)
		if err != nil {
			log.Printf("Unable to create ingress rule for service "+svc.ObjectMeta.Name, err)
		}
	case route:
		log.Println("Not yet implemented")
	case nodePort:
		useNodePort(ns, svc, c)

	case loadBalancer:
		useLoadBalancer(ns, svc, c)

	default:
		log.Fatalf("No match for [" + environment.Data[exposeRule] + "] expose-rule found.  Was the exposecontroller namespace setup by gofabric8?")
	}
}

func deleteExposeRule(svc string, c *client.Client) error {

	ns := strings.Split(svc, "/")[0]
	name := strings.Split(svc, "/")[1]

	rapi.AddToScheme(kapi.Scheme)
	rapiv1.AddToScheme(kapi.Scheme)

	ingressClient := c.Extensions().Ingress(ns)
	err := ingressClient.Delete(name, nil)
	if err != nil {
		log.Printf("Failed to delete ingress in namespace %s with error %v", ns, err)
		return err
	}

	success("Deleted ingress rule [" + name + "] in namespace [" + ns + "]")
	return nil
}

func useNodePort(ns string, svc *api.Service, c *client.Client) error {
	serviceLabels := svc.ObjectMeta.Labels
	if serviceLabels["expose"] == "true" {
		svc.Spec.Type = api.ServiceTypeNodePort
		svc, err := c.Services(ns).Update(svc)
		if err != nil {
			log.Printf("Unable to update service ["+svc.ObjectMeta.Name+"] with NodePort", err)
			return err
		}
		success("Exposed service [" + svc.ObjectMeta.Name + "] using NodePort")
	}
	log.Printf("Skipping service [" + svc.ObjectMeta.Name + "]")
	return nil
}

func useLoadBalancer(ns string, svc *api.Service, c *client.Client) error {
	serviceLabels := svc.ObjectMeta.Labels
	if serviceLabels["expose"] == "true" {
		svc.Spec.Type = api.ServiceTypeLoadBalancer
		svc, err := c.Services(ns).Update(svc)
		if err != nil {
			log.Printf("Unable to update service ["+svc.ObjectMeta.Name+"] with LoadBalancer", err)
			return err
		}
		success("Exposed service [" + svc.ObjectMeta.Name + "] using LoadBalancer. This can take a few minutes to ve create by cloud provider")
	}
	log.Printf("Skipping service [" + svc.ObjectMeta.Name + "]")
	return nil
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
	serviceLabels := service.ObjectMeta.Labels

	found := false

	if serviceLabels["expose"] == "true" {
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
				success("Exposed service " + name + " using ingress rule")
			}
		}
	} else {
		log.Printf("Skipping service [" + name + "]")
	}
	return nil
}

func success(msg string) {
	ct.ChangeColor(ct.Green, false, ct.None, false)
	log.Printf(msg)
	ct.ResetColor()
}
