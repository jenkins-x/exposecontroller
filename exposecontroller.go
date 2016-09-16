package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	kubectlutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"

	"github.com/fabric8io/exposecontroller/controller"
	"github.com/fabric8io/exposecontroller/version"
)

const (
	healthPort = 10254
)

var (
	flags = pflag.NewFlagSet("", pflag.ExitOnError)

	configMap = flags.String("configmap", "exposecontroller",
		`Name of the ConfigMap that contains the exposecontroller configuration to use`)

	resyncPeriod = flags.Duration("sync-period", 30*time.Second,
		`Relist and confirm services this often.`)

	watchNamespace = flags.String("controller-namespace", api.NamespaceAll,
		`Namespace to watch for Services. Default is to watch all namespaces`)

	healthzPort = flags.Int("healthz-port", healthPort, "port for healthz endpoint.")

	profiling = flags.Bool("profiling", true, `Enable profiling via web interface host:port/debug/pprof/`)
)

func main() {
	flags.AddGoFlagSet(flag.CommandLine)
	flags.Parse(os.Args)
	clientConfig := kubectlutil.DefaultClientConfig(flags)

	glog.Infof("Using build: %v", version.Version)

	config, err := clientConfig.ClientConfig()
	if err != nil {
		glog.Fatalf("error connecting to the client: %v", err)
	}
	kubeClient, err := client.New(config)

	if err != nil {
		glog.Fatalf("failed to create client: %v", err)
	}

	c, err := controller.NewController(kubeClient, *resyncPeriod, *watchNamespace, *configMap)
	if err != nil {
		glog.Fatalf("%v", err)
	}

	go registerHandlers()
	go handleSigterm(c)

	c.Run()
}

func registerHandlers() {
	mux := http.NewServeMux()

	if *profiling {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%v", *healthzPort),
		Handler: mux,
	}
	glog.Fatal(server.ListenAndServe())
}

func handleSigterm(c *controller.Controller) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-signalChan
	glog.Infof("Received %s, shutting down", sig)
	c.Stop()
}
