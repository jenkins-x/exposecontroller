CHART_REPO := http://jenkins-x-chartmuseum:8080
NAME := exposecontroller
OS := $(shell uname)
VERSION := $(shell cat ../../version/VERSION)
CHARTMUSEUM_CREDS_USR := $(shell cat /builder/home/basic-auth-user 2> /dev/null)
CHARTMUSEUM_CREDS_PSW := $(shell cat /builder/home/basic-auth-pass 2> /dev/null)

setup:
	minikube addons enable ingress
	brew install kubernetes-helm
	helm init

build: clean
	helm dependency build
	helm lint

install: clean build
	helm install . --name ${NAME}
	watch kubectl get pods

upgrade: clean build
	helm upgrade ${NAME} .
	watch kubectl get pods

delete:
	helm delete --purge ${NAME}

clean:
	rm -rf charts
	rm -rf ${NAME}*.tgz
	rm -rf requirements.lock

release: clean build
	helm package .
	curl --fail -u $(CHARTMUSEUM_CREDS_USR):$(CHARTMUSEUM_CREDS_PSW) --data-binary "@$(NAME)-$(VERSION).tgz" $(CHART_REPO)/api/charts
	rm -rf ${NAME}*.tgz