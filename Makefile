# Copyright (C) 2016 Red Hat, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Use the native vendor/ dependency system
export GO15VENDOREXPERIMENT=1

VERSION ?= $(shell cat version/VERSION)
REVISION=$(shell git rev-parse --short HEAD 2> /dev/null || echo 'unknown')
BRANCH=$(shell git rev-parse --abbrev-ref HEAD 2> /dev/null || echo 'unknown')
HOST=$(shell hostname -f)
BUILD_DATE=$(shell date +%Y%m%d-%H:%M:%S)
GO_VERSION=$(shell go version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
BUILD_DIR ?= ./out
ORG := github.com/fabric8io
REPOPATH ?= $(ORG)/exposecontroller
ROOT_PACKAGE := $(shell go list .)

ORIGINAL_GOPATH := $(GOPATH)
GOPATH := $(shell pwd)/_gopath

BUILDFLAGS := -ldflags \
  " -X $(ROOT_PACKAGE)/version.Version='$(VERSION)'\
    -X $(ROOT_PACKAGE)/version.Revision='$(REVISION)'\
    -X $(ROOT_PACKAGE)/version.Branch='$(BRANCH)'\
    -X $(ROOT_PACKAGE)/version.BuildUser='${USER}@$(HOST)'\
    -X $(ROOT_PACKAGE)/version.BuildDate='$(BUILD_DATE)'\
    -X $(ROOT_PACKAGE)/version.GoVersion='$(GO_VERSION)'\
    -s -w -extldflags '-static'"

GOFILES := go list  -f '{{join .Deps "\n"}}' $(REPOPATH) | grep $(REPOPATH) | xargs go list -f '{{ range $$file := .GoFiles }} {{$$.Dir}}/{{$$file}}{{"\n"}}{{end}}'
GOPACKAGES := $(shell go list ./... | grep -v /vendor/)

.PHONY: install
install: $(ORIGINAL_GOPATH)/bin/exposecontroller

$(ORIGINAL_GOPATH)/bin/exposecontroller: out/exposecontroller-$(GOOS)-$(GOARCH)
	cp $(BUILD_DIR)/exposecontroller-$(GOOS)-$(GOARCH) $(ORIGINAL_GOPATH)/bin/exposecontroller

out/exposecontroller: out/exposecontroller-$(GOOS)-$(GOARCH)
	cp $(BUILD_DIR)/exposecontroller-$(GOOS)-$(GOARCH) $(BUILD_DIR)/exposecontroller

out/exposecontroller-darwin-amd64: gopath $(shell $(GOFILES)) version/VERSION
	CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build $(BUILDFLAGS) -o $(BUILD_DIR)/exposecontroller-darwin-amd64 $(ROOT_PACKAGE)

out/exposecontroller-linux-amd64: gopath $(shell $(GOFILES)) version/VERSION
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build $(BUILDFLAGS) -o $(BUILD_DIR)/exposecontroller-linux-amd64 $(ROOT_PACKAGE)

out/exposecontroller-windows-amd64.exe: gopath $(shell $(GOFILES)) version/VERSION
	CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build $(BUILDFLAGS) -o $(BUILD_DIR)/exposecontroller-windows-amd64.exe $(ROOT_PACKAGE)

out/exposecontroller-linux-arm: gopath $(shell $(GOFILES)) version/VERSION
	CGO_ENABLED=0 GOARCH=arm GOOS=linux go build $(BUILDFLAGS) -o $(BUILD_DIR)/exposecontroller-linux-arm $(ROOT_PACKAGE)
	
.PHONY: test
test: gopath
	go test -v $(GOPACKAGES)

.PHONY: release
release: clean test cross
	mkdir -p release
	cp out/exposecontroller-*-amd64* release
	cp out/exposecontroller-*-arm* release
	gh-release checksums sha256
	gh-release create fabric8io/exposecontroller $(VERSION) master v$(VERSION)

.PHONY: cross
cross: out/exposecontroller-linux-amd64 out/exposecontroller-darwin-amd64 out/exposecontroller-windows-amd64.exe out/exposecontroller-linux-arm

.PHONY: gopath
gopath: $(GOPATH)/src/$(ORG)

$(GOPATH)/src/$(ORG):
	mkdir -p $(GOPATH)/src/$(ORG)
	ln -s -f $(shell pwd) $(GOPATH)/src/$(ORG)


.PHONY: clean
clean:
	rm -rf $(GOPATH)
	rm -rf $(BUILD_DIR)
	rm -rf release

.PHONY: docker
docker: out/exposecontroller-linux-amd64
	docker build -t "fabric8/exposecontroller:dev" .
