package exposestrategy

import "k8s.io/kubernetes/pkg/api"

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
