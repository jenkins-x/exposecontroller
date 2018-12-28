package exposestrategy

import (
	"reflect"
	"testing"

	"k8s.io/kubernetes/pkg/api"
)

func TestAddServiceAnnotationWithProtocol(t *testing.T) {
	tests := []struct {
		svc                 *api.Service
		hostName            string
		protocol            string
		expectedAnnotations map[string]string
	}{
		{
			svc:      &api.Service{},
			hostName: "example.com",
			protocol: "http",
			expectedAnnotations: map[string]string{
				ExposeAnnotationKey: "http://example.com",
			},
		},
		{
			svc:      &api.Service{},
			hostName: "example.com",
			protocol: "https",
			expectedAnnotations: map[string]string{
				ExposeAnnotationKey: "https://example.com",
			},
		},
		{
			svc: &api.Service{
				ObjectMeta: api.ObjectMeta{
					Annotations: map[string]string{
						ApiServicePathAnnotationKey: "some/path",
					},
				},
			},
			hostName: "example.com",
			protocol: "https",
			expectedAnnotations: map[string]string{
				ApiServicePathAnnotationKey: "some/path",
				ExposeAnnotationKey:         "https://example.com/some/path",
			},
		},
		{
			svc: &api.Service{
				ObjectMeta: api.ObjectMeta{
					Annotations: map[string]string{
						ExposeHostNameAsAnnotationKey: "osiris.deislabs.io/ingressHostname",
					},
				},
			},
			hostName: "example.com",
			protocol: "http",
			expectedAnnotations: map[string]string{
				ExposeHostNameAsAnnotationKey:        "osiris.deislabs.io/ingressHostname",
				"osiris.deislabs.io/ingressHostname": "example.com",
				ExposeAnnotationKey:                  "http://example.com",
			},
		},
	}

	for i, test := range tests {
		svc, err := addServiceAnnotationWithProtocol(test.svc, test.hostName, test.protocol)
		if err != nil {
			t.Errorf("[%d] got unexpected error: %v", i, err)
			continue
		}

		if !reflect.DeepEqual(test.expectedAnnotations, svc.Annotations) {
			t.Errorf("[%d] Got the following annotations %#v but expected %#v", i, svc.Annotations, test.expectedAnnotations)
		}
	}
}

func TestRemoveServiceAnnotation(t *testing.T) {
	tests := []struct {
		svc                 *api.Service
		expectedAnnotations map[string]string
	}{
		{
			svc:                 &api.Service{},
			expectedAnnotations: nil,
		},
		{
			svc: &api.Service{
				ObjectMeta: api.ObjectMeta{
					Annotations: map[string]string{
						ExposeAnnotationKey: "http://example.com",
						"some-key":          "some value",
					},
				},
			},
			expectedAnnotations: map[string]string{
				"some-key": "some value",
			},
		},
		{
			svc: &api.Service{
				ObjectMeta: api.ObjectMeta{
					Annotations: map[string]string{
						ExposeHostNameAsAnnotationKey:        "osiris.deislabs.io/ingressHostname",
						"osiris.deislabs.io/ingressHostname": "example.com",
						ApiServicePathAnnotationKey:          "some/path",
						ExposeAnnotationKey:                  "http://example.com/some/path",
					},
				},
			},
			expectedAnnotations: map[string]string{
				ExposeHostNameAsAnnotationKey: "osiris.deislabs.io/ingressHostname",
				ApiServicePathAnnotationKey:   "some/path",
			},
		},
	}

	for i, test := range tests {
		svc := removeServiceAnnotation(test.svc)
		if !reflect.DeepEqual(test.expectedAnnotations, svc.Annotations) {
			t.Errorf("[%d] Got the following annotations %#v but expected %#v", i, svc.Annotations, test.expectedAnnotations)
		}
	}
}
