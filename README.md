# Exposer

Automatically exposes services creating ingress routes using the namespace as the wildcard

# Future

On startup it would be good to check if an ingress controller is already running in the cluster, if not create one in an appropriate namespace using a `nodeselector` that chooses a node with a public ip.


## Building

 * install [go version 1.5.1 or later](https://golang.org/doc/install)
 * install [glide](https://github.com/Masterminds/glide#install)
 * type the following:
 * when using local kubernetes VM export the `DOCKER_HOST` env var to build the exposer image and run inside kubernetes

```
cd $GOPATH
mkdir -p src/github.com/fabric8io/
cd src/github.com/fabric8io/
git clone https://github.com/fabric8io/exposer.git
cd exposer

make bootstrap
```

 * then to build the binary

     `make build` (not currently working on OSX so use `GOOS=linux GOARCH=386 go build -o bin/exposer exposer.go`)

 * build docker image

     `docker build -t fabric8/exposer:0.1 .`

 * run in kubernetes

     `kubectl run exposer --image fabric8/exposer:0.1`

## Releasing

Just run `make release`. This will cross-compile for all supported platforms, create tag & upload tarballs (zip file for Windows) to Github releases for download.

Updating the version is done via `make bump` to bump minor version & `make bump-patch` to bump patch version. This is necessary as tags are created from the version specified when releasing.
