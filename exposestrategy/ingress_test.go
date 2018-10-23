package exposestrategy_test

import (
	"testing"

	"github.com/jenkins-x/exposecontroller/exposestrategy"
	"gotest.tools/assert"
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
	assert.Equal(t, rootPath+"/test", path)

	rootPath = "/root"
	path = es.PortPath(svc, port, rootPath)
	assert.Equal(t, rootPath+"/test", path)

	rootPath = "/root/"
	path = es.PortPath(svc, port, rootPath)
	assert.Equal(t, rootPath+"test", path)
}
