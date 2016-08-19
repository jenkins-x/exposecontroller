# exposecontroller

Automatically expose services creating ingress rules, openshift routes or modifying services to use kubernetes nodePort or loadBalancer service types

## Setup

___NOTE___ if you have used [gofabric8](https://github.com/fabric8io/gofabric8) you can skip this and go straight to [Run](#run)

As we create resources via the Kubernetes API server we'll need to create a Service Account.

### Kubernetes 

```
cat <<EOF | kubectl create -f -
apiVersion: "v1"
kind: "ServiceAccount"
metadata:
  labels:
    provider: "fabric8"
    project: "exposecontroller"
  name: "exposecontroller"
EOF
```

### OpenShift

```
cat <<EOF | oc create -f -
apiVersion: "v1"
kind: "ServiceAccount"
metadata:
  labels:
    provider: "fabric8"
    project: "exposecontroller"
  name: "exposecontroller"
EOF
``` 

You will also need to grant the correct roles

    oc adm policy add-cluster-role-to-user cluster-admin system:serviceaccount:default:exposecontroller
    oc adm policy add-cluster-role-to-group cluster-reader system:serviceaccounts # probably too open for all setups

## Configure

If you're not using [gofabric8](https://github.com/fabric8io/gofabric8) to setup your environment then you'll need to create a `configmap` in oder to specify the approach `exposecontroller` will use to configure accessing your services.

When using either Kubernetes Ingress or OpenShift Routes you will need to set the domain that you've used with your DNS provider (fabric8 uses [cloudflare](https://www.cloudflare.com))

You also need to specify an `expose-rule` type that you want the __exposecontroller__ to use.

### types
- `ingress` - Kubernetes Ingress [see](http://kubernetes.io/docs/user-guide/ingress/)
- `load-balancer` - Cloud provider external loadbalancer [see](http://kubernetes.io/docs/user-guide/load-balancer/)
- `node-port` - Recomended for local development using minikube / minishift without Ingress or Router running [see](http://kubernetes.io/docs/user-guide/services/#type-nodeport)
- `route` - OpenShift Route [see](https://docs.openshift.com/enterprise/3.2/dev_guide/routes.html)

### example

### Kubernetes
```
cat <<EOF | kubectl create -f -
apiVersion: "v1"
data:
  expose-rule: "ingress"
  domain: "replace.me.io"
  watch-rate-milliseconds: "5000"
kind: "ConfigMap"
metadata:
  name: "exposecontroller"
EOF
```

### OpenShift

```
cat <<EOF | oc create -f -
apiVersion: "v1"
data:
  expose-rule: "route"
  domain: "replace.me.io"
  watch-rate-milliseconds: "5000"
kind: "ConfigMap"
metadata:
  name: "exposecontroller"
EOF
```

## Run

We're adding install via Helm however until then...

### Kubernetes

```
kc create -f http://central.maven.org/maven2/io/fabric8/devops/apps/exposecontroller/2.2.236/exposecontroller-2.2.236-kubernetes.yml
```

### OpenShift
```
oc create -f http://central.maven.org/maven2/io/fabric8/devops/apps/exposecontroller/2.2.236/exposecontroller-2.2.236-openshift.yml
```

## Label

Now label your service with `expose=true` in [CD Pipelines](https://blog.fabric8.io/create-and-explore-continuous-delivery-pipelines-with-fabric8-and-jenkins-on-openshift-661aa82cb45a#.lx020ys70) or with CLI...

```
kubectl label svc foo expose=true
```

__exposecontroller__ will use your `expose-rule` in the configmap above to automatically watch for new services and create ingress / routes / nodeports / loadbalacers for you.

## Building

 * install [go version 1.5.1 or later](https://golang.org/doc/install)
 * install [glide](https://github.com/Masterminds/glide#install)
 * type the following:
 * when using minikube or minishift expose the docker daemon to build the __exposecontroller__ image and run inside kubernetes.  e.g  `export DOCKER_API_VERSION=1.23 && eval $(minikube docker-env)`

```
cd $GOPATH
mkdir -p src/github.com/fabric8io/
cd src/github.com/fabric8io/
git clone https://github.com/fabric8io/exposecontroller.git
cd exposecontroller

make bootstrap
```

### Run locally

After setting some test env vars you'll need to build the binary and run it.  You may need to copy a token and cert from a pod to you local filesystem under `/var/run/secrets/kubernetes.io/serviceaccount/`.

    export KUBERNETES_SERVICE_HOST=192.168.99.100
    export KUBERNETES_SERVICE_PORT=443
    export KUBERNETES_NAMESPACE=default
    rm -rf bin/exposecontroller && make && ./bin/exposecontroller


### Run on Kubernetes or OpenShift

 * build the binary

    `make` 
     
    (currently not currently working on OSX so use)
     
    `GOOS=linux GOARCH=386 go build -o bin/exposecontroller exposecontroller.go`

 * build docker image

     `docker build -t fabric8/exposecontroller:test .`

 * run in kubernetes

     `kubectl run exposecontroller --image fabric8/exposecontroller:test `

# Future

On startup it would be good to check if an ingress controller is already running in the cluster, if not create one in an appropriate namespace using a `nodeselector` that chooses a node with a public ip.
