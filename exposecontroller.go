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
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fabric8io/exposecontroller/client"
	"github.com/fabric8io/exposecontroller/util"
	osclient "github.com/openshift/origin/pkg/client"
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
	domain              = "domain"
	exposeAnnotationKey = "fabric8.io/exposeUrl"
	exposeRule          = "expose-rule"
	exposeControllerCM  = "exposecontroller"
	ingress             = "ingress"
	loadBalancer        = "load-balancer"
	nodePort            = "node-port"
	route               = "route"
	exposeLabel         = "expose=true"
	watchRate           = "watch-rate-milliseconds"
)

func main() {

	f := cmdutil.NewFactory(nil)
	c, cfg := client.NewClient(f)
	oc, _ := client.NewOpenShiftClient(cfg)

	util.Successf("Connected")

	var err error
	currentNs := os.Getenv("KUBERNETES_NAMESPACE")

	if currentNs == "" {
		currentNs, _, err = f.DefaultNamespace()
		if err != nil {
			util.Error("No $KUBERNETES_NAMESPACE environment variable set")
		}
	}

	resyncPeriod := getResyncPeriod(c, currentNs)
	log.Printf("ResyncPeriod is %v", resyncPeriod)

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
		resyncPeriod,
		framework.ResourceEventHandlerFuncs{
			AddFunc:    serviceAdded(c, oc, currentNs),
			UpdateFunc: serviceUpdated(c, oc, currentNs),
			DeleteFunc: serviceDeleted(c, oc, currentNs),
		},
	)
	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(stop)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getResyncPeriod(c *kclient.Client, currentNs string) time.Duration {
	environment, err := c.ConfigMaps(currentNs).Get(exposeControllerCM)
	if err != nil {
		log.Fatalf("No ConfigMap with name %s found in namespace %s.  Was the exposecontroller namespace setup by gofabric8? %v", exposeControllerCM, currentNs, err)
	}

	period, ok := environment.Data[watchRate]
	if ok {
		milliseconds, err := time.ParseDuration(period + "ms")
		if err != nil {
			log.Printf("Error parsing %v : %v", period, err)
		}
		return milliseconds
	}
	return time.Millisecond * 5000 // default of 5 seconds
}

func serviceAdded(c *kclient.Client, oc *osclient.Client, currentNs string) func(obj interface{}) {
	return func(obj interface{}) {
		svc := obj.(*api.Service)
		addExposeRule(c, oc, svc, currentNs)
	}
}

func serviceUpdated(c *kclient.Client, oc *osclient.Client, currentNs string) func(oldObj interface{}, newObj interface{}) {
	return func(oldObj interface{}, newObj interface{}) {
		exposeLabelKey, exposeLabelValue := getExposeLabel()
		oldSvc := oldObj.(*api.Service)
		oldServiceLabels := oldSvc.ObjectMeta.Labels

		newSvc := newObj.(*api.Service)
		newServiceLabels := newSvc.ObjectMeta.Labels

		if oldValue, oldFound := oldServiceLabels[exposeLabelKey]; oldFound {
			if newValue, newFound := newServiceLabels[exposeLabelKey]; !newFound {
				// delete
				deleteExposeRule(newSvc.Namespace, newSvc.ObjectMeta.Name, c, oc, currentNs)
			} else {
				// if the expose label has changed
				if oldValue != newValue {
					if newValue == exposeLabelValue {
						// add
						addExposeRule(c, oc, newSvc, currentNs)
					} else {
						// delete
						deleteExposeRule(newSvc.Namespace, newSvc.ObjectMeta.Name, c, oc, currentNs)
					}
				}
			}
		} else if newValue, newFound := newServiceLabels[exposeLabelKey]; newFound {
			// if the expose label has changed
			if oldValue != newValue {
				if newValue == exposeLabelValue {
					// add
					addExposeRule(c, oc, newSvc, currentNs)
				} else {
					// delete
					deleteExposeRule(newSvc.Namespace, newSvc.ObjectMeta.Name, c, oc, currentNs)
				}
			}
		}
	}
}

func serviceDeleted(c *kclient.Client, oc *osclient.Client, currentNs string) func(obj interface{}) {
	return func(obj interface{}) {
		svc, ok := obj.(cache.DeletedFinalStateUnknown)
		if ok {
			// service key is in the form namespace/name
			ns := strings.Split(svc.Key, "/")[0]
			name := strings.Split(svc.Key, "/")[1]
			deleteExposeRule(ns, name, c, oc, currentNs)
		} else {
			svc, ok := obj.(*api.Service)
			if ok {
				deleteExposeRule(svc.Namespace, svc.ObjectMeta.Name, c, oc, currentNs)
			} else {
				log.Fatalf("Error getting details of deleted service")
			}
		}
	}
}

func addExposeRule(c *kclient.Client, oc *osclient.Client, svc *api.Service, currentNs string) {
	log.Printf("Found service %s in namespace %s", svc.ObjectMeta.Name, svc.Namespace)

	environment, err := c.ConfigMaps(currentNs).Get(exposeControllerCM)
	if err != nil {
		log.Fatalf("No ConfigMap with name %s found in namespace %s.  Was the exposecontroller namespace setup by gofabric8? %v", exposeControllerCM, currentNs, err)
	}

	d, ok := environment.Data[domain]
	if !ok {
		log.Fatalf("No ConfigMap data with name %s found in namespace %s.  Was the exposecontroller namespace setup by gofabric8? %v", domain, currentNs, err)
	}

	switch environment.Data[exposeRule] {
	case ingress:
		if util.TypeOfMaster(c) == util.OpenShift {
			log.Println("Ingress is not currently supported on OpenShift, please use Routes")
		} else {
			err := createIngress(svc.Namespace, d, svc, c)
			if err != nil {
				log.Printf("Unable to create ingress rule for service %s %v", svc.ObjectMeta.Name, err)
			}
		}

	case route:
		if util.TypeOfMaster(c) != util.OpenShift {
			log.Println("Routes are only available on OpenShift, please use Ingress")
		} else {
			createRoute(svc.Namespace, d, svc, c, oc)
		}
	case nodePort:
		useNodePort(svc.Namespace, svc, c)

	case loadBalancer:
		useLoadBalancer(svc.Namespace, svc, c)

	default:
		log.Fatalf("No match for %s expose-rule found.  Was the exposecontroller namespace setup by gofabric8?", environment.Data[exposeRule])
	}
}

func deleteExposeRule(ns string, name string, c *kclient.Client, oc *osclient.Client, currentNs string) error {

	environment, err := c.ConfigMaps(currentNs).Get(exposeControllerCM)
	if err != nil {
		log.Fatalf("No ConfigMap with name %s found in namespace %s.  Was the exposecontroller namespace setup by gofabric8? %v", exposeControllerCM, currentNs, err)
	}

	switch environment.Data[exposeRule] {
	case ingress:
		return deleteIngress(ns, name, c)

	case route:
		return deleteRoute(ns, name, oc)

	case nodePort:
		return nil

	case loadBalancer:
		return nil

	default:
		log.Fatalf("No match for %s expose-rule found.  Was the exposecontroller namespace setup by gofabric8?", environment.Data[exposeRule])
	}

	return nil
}

func deleteIngress(ns string, name string, c *kclient.Client) error {
	rapi.AddToScheme(kapi.Scheme)
	rapiv1.AddToScheme(kapi.Scheme)

	ingressClient := c.Extensions().Ingress(ns)
	err := ingressClient.Delete(name, nil)
	if err != nil {
		log.Printf("Failed to delete ingress in namespace %s with error %v", ns, err)
		return err
	}

	util.Successf("Deleted ingress rule %s in namespace %s", name, ns)
	return nil
}

func deleteRoute(ns string, name string, c *osclient.Client) error {

	rapi.AddToScheme(kapi.Scheme)
	rapiv1.AddToScheme(kapi.Scheme)

	err := c.Routes(ns).Delete(name)
	if err != nil {
		log.Printf("Failed to delete openshift route %s in namespace %s with error %v", name, ns, err)
		return err
	}

	util.Successf("Deleted openshift route %s in namespace %s", name, ns)
	return nil
}

func useNodePort(ns string, svc *api.Service, c *kclient.Client) error {
	serviceLabels := svc.ObjectMeta.Labels
	exposeLabelKey, exposeLabelValue := getExposeLabel()
	if serviceLabels[exposeLabelKey] == exposeLabelValue {
		svc.Spec.Type = api.ServiceTypeNodePort
		svc, err := c.Services(ns).Update(svc)
		if err != nil {
			log.Printf("Unable to update service %s with NodePort %v", svc.ObjectMeta.Name, err)
			return err
		}

		if len(svc.Spec.Ports) > 1 {
			util.Warnf("Found %v ports %s", len(svc.Spec.Ports), svc.Name)

		}

		for _, port := range svc.Spec.Ports {
			nodePort := strconv.Itoa(port.NodePort)
			hostName := ":" + nodePort
			addServiceAnnotation(c, ns, svc, hostName)
		}

		util.Successf("Exposed service %s using NodePort", svc.ObjectMeta.Name)
	} else {
		log.Printf("Skipping service %s", svc.ObjectMeta.Name)
	}
	return nil
}

func useLoadBalancer(ns string, svc *api.Service, c *kclient.Client) error {
	serviceLabels := svc.ObjectMeta.Labels
	exposeLabelKey, exposeLabelValue := getExposeLabel()
	if serviceLabels[exposeLabelKey] == exposeLabelValue {
		svc.Spec.Type = api.ServiceTypeLoadBalancer
		svc, err := c.Services(ns).Update(svc)
		if err != nil {
			log.Printf("Unable to update service %s with LoadBalancer %v", svc.ObjectMeta.Name, err)
			return err
		}
		hostName := svc.Spec.LoadBalancerIP
		if hostName != "" {
			addServiceAnnotation(c, ns, svc, hostName)
		}
		util.Successf("Exposed service %s using LoadBalancer. This can take a few minutes to be create by cloud provider", svc.ObjectMeta.Name)
	} else {
		log.Printf("Skipping service %s", svc.ObjectMeta.Name)
	}

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

	exposeLabelKey, exposeLabelValue := getExposeLabel()
	if serviceLabels[exposeLabelKey] == exposeLabelValue {
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
				addServiceAnnotation(c, ns, service, hostName)
				util.Successf("Exposed service %s using ingress rule", name)
			}
		}
	} else {
		log.Printf("Skipping service %s", name)
	}
	return nil
}

func addServiceAnnotation(c *kclient.Client, ns string, svc *api.Service, hostName string) {

	svc.Annotations[exposeAnnotationKey] = hostName
	_, err := c.Services(ns).Update(svc)
	if err != nil {
		util.Errorf("Failed to add the %s to service %s %v", exposeAnnotationKey, svc.Name, err)
	}
}

func createRoute(ns string, domain string, svc *api.Service, c *kclient.Client, oc *osclient.Client) error {

	rapi.AddToScheme(kapi.Scheme)
	rapiv1.AddToScheme(kapi.Scheme)

	name := svc.ObjectMeta.Name

	var labels = make(map[string]string)
	labels["provider"] = "fabric8"

	serviceLabels := svc.ObjectMeta.Labels
	exposeLabelKey, exposeLabelValue := getExposeLabel()
	if serviceLabels[exposeLabelKey] == exposeLabelValue {
		if name != "kubernetes" {
			routes := oc.Routes(ns)
			_, err := routes.Get(name)
			if err != nil {
				// need to add namespace back in the hostname but we have to update the fabric8-console oauthclient too
				// see https://github.com/fabric8io/gofabric8/issues/98
				//hostName := name + "." + ns + "." + domain
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
				addServiceAnnotation(c, ns, svc, hostName)
				util.Successf("Exposed service %s using openshift route", name)
			}
		}
	} else {
		log.Printf("Skipping service %s", name)
	}
	return nil
}

func getExposeLabel() (string, string) {
	key := strings.Split(exposeLabel, "=")[0]
	value := strings.Split(exposeLabel, "=")[1]
	return key, value
}
