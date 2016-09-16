package controller

import (
	"time"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/client/record"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

type Controller struct {
	client *client.Client

	svcController *framework.Controller
	svcLister     cache.StoreToServiceLister

	configMap string

	recorder record.EventRecorder

	stopCh chan struct{}
}

func NewController(
	kubeClient *client.Client,
	resyncPeriod time.Duration, namespace, configMapName string) (*Controller, error) {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(kubeClient.Events(namespace))

	c := Controller{
		client:    kubeClient,
		stopCh:    make(chan struct{}),
		configMap: configMapName,
		recorder: eventBroadcaster.NewRecorder(api.EventSource{
			Component: "expose-controller",
		}),
	}

	c.svcLister.Store, c.svcController = framework.NewInformer(
		&cache.ListWatch{
			ListFunc:  serviceListFunc(c.client, namespace),
			WatchFunc: serviceWatchFunc(c.client, namespace),
		},
		&api.Service{}, resyncPeriod, framework.ResourceEventHandlerFuncs{})

	return &c, nil
}

// Run starts the controller.
func (c *Controller) Run() {
	glog.Infof("starting expose controller")

	go c.svcController.Run(c.stopCh)

	<-c.stopCh
}

func (c *Controller) Stop() {
	glog.Infof("stopping expose controller")

	close(c.stopCh)
}

func serviceListFunc(c *client.Client, ns string) func(api.ListOptions) (runtime.Object, error) {
	return func(opts api.ListOptions) (runtime.Object, error) {
		return c.Services(ns).List(opts)
	}
}

func serviceWatchFunc(c *client.Client, ns string) func(options api.ListOptions) (watch.Interface, error) {
	return func(options api.ListOptions) (watch.Interface, error) {
		return c.Services(ns).Watch(options)
	}
}
