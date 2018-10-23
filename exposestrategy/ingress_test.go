package exposestrategy_test

import (
	"testing"

	"github.com/jenkins-x/exposecontroller/exposestrategy"
	"k8s.io/kubernetes/pkg/api"
)

func TestPortPath(t *testing.T) {
	es := &exposestrategy.IngressStrategy{}
	port := &api.ServicePort{
		Name: "test",
	}
	svc := &api.Service{
		Spec: api.ServiceSpec{
			Ports: []api.ServicePort{*port, *port},
		},
	}

	rootPath := ""
	path := es.PortPath(svc, port, rootPath)
	assertStringEquals(t, rootPath+"/test", path, "port path")

	rootPath = "/root"
	path = es.PortPath(svc, port, rootPath)
	assertStringEquals(t, rootPath+"/test", path, "port path")

	rootPath = "/root/"
	path = es.PortPath(svc, port, rootPath)
	assertStringEquals(t, rootPath+"test", path, "port path")
}

func assertStringEquals(t *testing.T, expected, actual, message string) {
	if expected != actual {
		t.Errorf("%s was not equal. Expected %s but got %s\n", message, expected, actual)
	}
}
