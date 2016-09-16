package exposestrategy

import (
	"bytes"
	"net"

	"github.com/pkg/errors"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/strategicpatch"
)

type ExposeStrategy interface {
	Add(svc *api.Service) error
	Remove(svc *api.Service) error
}

type Label struct {
	Key   string
	Value string
}

var (
	ExposeLabel         = Label{Key: "expose", Value: "true"}
	ExposeAnnotationKey = "fabric8.io/exposeUrl"
)

func addServiceAnnotation(svc *api.Service, hostName string) (*api.Service, error) {
	// default to http
	protocol := "http"

	// if a port is on the hostname check is its a default http / https port
	_, port, err := net.SplitHostPort(hostName)
	if err == nil {
		if port == "443" || port == "8443" {
			protocol = "https"
		} else {
			// check if the service port has a name of https
			for _, port := range svc.Spec.Ports {
				if port.Name == "https" {
					protocol = port.Name
				}
			}
		}
	}
	exposeURL := protocol + "://" + hostName
	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}
	svc.Annotations[ExposeAnnotationKey] = exposeURL

	return svc, nil
}

func removeServiceAnnotation(svc *api.Service) *api.Service {
	delete(svc.Annotations, ExposeAnnotationKey)

	return svc
}

func createPatch(a runtime.Object, b runtime.Object, encoder runtime.Encoder, dataStruct interface{}) ([]byte, error) {
	var aBuf bytes.Buffer
	err := encoder.Encode(a, &aBuf)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to encode object: %v", a)
	}
	var bBuf bytes.Buffer
	err = encoder.Encode(b, &bBuf)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to encode object: %v", b)
	}

	aBytes := aBuf.Bytes()
	bBytes := bBuf.Bytes()

	if bytes.Compare(aBytes, bBytes) == 0 {
		return nil, nil
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(aBytes, bBytes, dataStruct)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create patch")
	}

	return patch, nil
}
