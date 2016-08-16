/**
 * Copyright (C) 2015 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *         http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/daviddengcn/go-colortext"
	"github.com/fabric8io/exposecontroller/client"
	"github.com/fabric8io/exposecontroller/util"
	rapi "github.com/openshift/origin/pkg/route/api"
	rapiv1 "github.com/openshift/origin/pkg/route/api/v1"
	"k8s.io/kubernetes/pkg/api"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/controller/framework"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
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

	f := cmdutil.NewFactory(nil)
	c, _ := client.NewClient(f)

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

func serviceAdded(c *kclient.Client) func(obj interface{}) {
	return func(obj interface{}) {
		svc := obj.(*api.Service)
		addExposeRule(c, svc, svc.Namespace)
	}
}

func serviceDeleted(c *kclient.Client) func(obj interface{}) {
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

func addExposeRule(c *kclient.Client, oc *osclient.Client, svc *api.Service, ns string) {
	log.Printf("Found service %s in namespace %s", svc.ObjectMeta.Name, ns)
	currentNs := os.Getenv("KUBERNETES_NAMESPACE")
	if len(currentNs) <= 0 {
		log.Fatalf("No KUBERNETES_NAMESPACE env var set")
	}

	environment, err := c.ConfigMaps(currentNs).Get(fabric8Environment)
	if err != nil {
		log.Fatalf("No ConfigMap with name %s found in namespace %s.  Was the exposecontroller namespace setup by gofabric8? %v", fabric8Environment, currentNs, err)
	}

	d, ok := environment.Data[domain]
	if !ok {
		log.Fatalf("No ConfigMap data with name %s found in namespace %s.  Was the exposecontroller namespace setup by gofabric8? %v", domain, currentNs, err)
	}

	switch environment.Data[exposeRule] {
	case ingress:
		err := createIngress(ns, d, svc, c)
		if err != nil {
			log.Printf("Unable to create ingress rule for service %s %v", svc.ObjectMeta.Name, err)
		}
	case route:
		if util.TypeOfMaster(c) != util.OpenShift {
			log.Println("Routes are only available on OpenShift, please look at using Ingress")
		} else {
			createRoute(ns, d, svc, c, oc)
		}
	case nodePort:
		useNodePort(ns, svc, c)

	case loadBalancer:
		useLoadBalancer(ns, svc, c)

	default:
		log.Fatalf("No match for %s expose-rule found.  Was the exposecontroller namespace setup by gofabric8?", environment.Data[exposeRule])
	}
}

func deleteExposeRule(svc string, c *kclient.Client) error {

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

func useNodePort(ns string, svc *api.Service, c *kclient.Client) error {
	serviceLabels := svc.ObjectMeta.Labels
	if serviceLabels["expose"] == "true" {
		svc.Spec.Type = api.ServiceTypeNodePort
		svc, err := c.Services(ns).Update(svc)
		if err != nil {
			log.Printf("Unable to update service %s with NodePort %v", svc.ObjectMeta.Name, err)
			return err
		}
		successf("Exposed service %s using NodePort", svc.ObjectMeta.Name)
	}
	log.Printf("Skipping service %s", svc.ObjectMeta.Name)
	return nil
}

func useLoadBalancer(ns string, svc *api.Service, c *kclient.Client) error {
	serviceLabels := svc.ObjectMeta.Labels
	if serviceLabels["expose"] == "true" {
		svc.Spec.Type = api.ServiceTypeLoadBalancer
		svc, err := c.Services(ns).Update(svc)
		if err != nil {
			log.Printf("Unable to update service %s with LoadBalancer %v", svc.ObjectMeta.Name, err)
			return err
		}
		successf("Exposed service %s using LoadBalancer. This can take a few minutes to be create by cloud provider", svc.ObjectMeta.Name)
	}
	log.Printf("Skipping service %s", svc.ObjectMeta.Name)
	return nil
}

func createIngress(ns string, domain string, service *api.Service, c *kclient.Client) error {
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
				successf("Exposed service %s using ingress rule", name)
			}
		}
	} else {
		log.Printf("Skipping service %s", name)
	}
	return nil
}

func createRoute(ns string, domain string, service *api.Service, c *kclient.Client, oc *osclient.Client) error {

	rapi.AddToScheme(kapi.Scheme)
	rapiv1.AddToScheme(kapi.Scheme)

	rc, err := c.Services(ns).List(kapi.ListOptions{})
	if err != nil {
		log.Printf("Failed to load services in namespace %s with error %v", ns, err)
		return err
	}
	var labels = make(map[string]string)
	labels["provider"] = "fabric8"

	items := rc.Items
	for _, service := range items {
		// TODO use the external load balancer as a way to know if we should create a route?
		name := service.ObjectMeta.Name
		if name != "kubernetes" {
			routes := oc.Routes(ns)
			_, err = routes.Get(name)
			if err != nil {
				hostName := name + "." + domain
				route := rapi.Route{
					ObjectMeta: kapi.ObjectMeta{
						Labels: labels,
						Name:   name,
					},
					Spec: rapi.RouteSpec{
						Host: hostName,
						To:   kapi.ObjectReference{Name: name},
					},
				}
				// lets create the route
				_, err = routes.Create(&route)
				if err != nil {
					log.Printf("Failed to create the route %s with error %v", name, err)
					return err
				}
			}
		}
	}
	return nil
}

// Successf prints success message
func successf(msg string, args ...interface{}) {
	success(fmt.Sprintf(msg, args...))
}

func success(msg string) {
	ct.ChangeColor(ct.Green, false, ct.None, false)
	log.Printf(msg)
	ct.ResetColor()
}
