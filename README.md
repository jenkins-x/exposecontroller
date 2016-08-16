# Exposecontroller

Automatically expose services creating ingress rules, openshift routes or modifying services to use kubernetes nodePort or loadBalancer service types

# Future

On startup it would be good to check if an ingress controller is already running in the cluster, if not create one in an appropriate namespace using a `nodeselector` that chooses a node with a public ip.


## Building

 * install [go version 1.5.1 or later](https://golang.org/doc/install)
 * install [glide](https://github.com/Masterminds/glide#install)
 * type the following:
 * when using minikube or minishift expose the docker daemon to build the exposecontroller image and run inside kubernetes.  e.g  `export DOCKER_API_VERSION=1.23 && eval $(minikube docker-env)`

```
cd $GOPATH
mkdir -p src/github.com/fabric8io/
cd src/github.com/fabric8io/
git clone https://github.com/fabric8io/exposecontroller.git
cd exposecontroller

make
```

 * then to build the binary

     `make build` (not currently working on OSX so use `GOOS=linux GOARCH=386 go build -o bin/exposecontroller exposecontroller.go`)

 * build docker image

     `docker build -t fabric8/exposecontroller:0.1 .`

 * run in kubernetes

     `kubectl run exposecontroller --image fabric8/exposecontroller:0.1`

## Releasing

Just run `make release`. This will cross-compile for all supported platforms, create tag & upload tarballs (zip file for Windows) to Github releases for download.

Updating the version is done via `make bump` to bump minor version & `make bump-patch` to bump patch version. This is necessary as tags are created from the version specified when releasing.
