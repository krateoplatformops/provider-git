# Set the shell to bash always
SHELL := /bin/bash

# Look for a .env file, and if present, set make variables from it.
ifneq (,$(wildcard ./.env))
	include .env
	export $(shell sed 's/=.*//' .env)
endif

KIND_CLUSTER_NAME ?= local-dev
KUBECONFIG ?= $(HOME)/.kube/config

VERSION := $(shell git describe --dirty --always --tags | sed 's/-/./2' | sed 's/-/./2')
ifndef VERSION
VERSION := 0.0.0
endif

BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
REPO_URL := $(shell git config --get remote.origin.url | sed "s/git@/https\:\/\//; s/\.com\:/\.com\//; s/\.git//")
LAST_COMMIT := $(shell git log -1 --pretty=%h)

PROJECT_NAME := provider-git
ORG_NAME := krateoplatformops
VENDOR := Kiratech

# Github Container Registry
DOCKER_REGISTRY := ghcr.io/$(ORG_NAME)

TARGET_OS := linux
TARGET_ARCH := amd64

# Tools
KIND=$(shell which kind)
LINT=$(shell which golangci-lint)
KUBECTL=$(shell which kubectl)
DOCKER=$(shell which docker)


.DEFAULT_GOAL := help

.PHONY: help
## help: Print this help
help: Makefile
	@echo
	@echo " Choose a command run in "$(PROJECT_NAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo


.PHONY: print.vars
## print.vars: Print all the build variables
print.vars:
	@echo VENDOR=$(VENDOR)
	@echo ORG_NAME=$(ORG_NAME)
	@echo PROJECT_NAME=$(PROJECT_NAME)
	@echo REPO_URL=$(REPO_URL)
	@echo LAST_COMMIT=$(LAST_COMMIT)
	@echo VERSION=$(VERSION)
	@echo BUILD_DATE=$(BUILD_DATE)
	@echo TARGET_OS=$(TARGET_OS)
	@echo TARGET_ARCH=$(TARGET_ARCH)
	@echo DOCKER_REGISTRY=$(DOCKER_REGISTRY)


.PHONY: dev
## dev: Run the controller in debug mode
dev: generate
	$(KUBECTL) apply -f package/crds/ -R
	go run cmd/provider/main.go -d

.PHONY: generate
## generate: Generate all CRDs
generate: tidy
	go generate ./...
	@find package/crds -name *.yaml -exec sed -i.sed -e '1,2d' {} \;
	@find package/crds -name *.yaml.sed -delete

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: test
test:
	go test -v ./...

.PHONY: lint
lint:
	$(LINT) run

.PHONY: kind.up
## kind.up: Starts a KinD cluster for local development
kind.up:
	@$(KIND) get kubeconfig --name $(KIND_CLUSTER_NAME) >/dev/null 2>&1 || $(KIND) create cluster --name=$(KIND_CLUSTER_NAME)

.PHONY: kind.down
## kind.down: Shuts down the KinD cluster
kind.down:
	@$(KIND) delete cluster --name=$(KIND_CLUSTER_NAME)

.PHONY: image.build
## image.build: Build the Docker image
image.build:
	@$(DOCKER) build -t "$(DOCKER_REGISTRY)/$(PROJECT_NAME):$(VERSION)" \
	--build-arg METRICS_PORT=9090 \
	--build-arg VERSION="$(VERSION)" \
	--build-arg BUILD_DATE="$(BUILD_DATE)" \
	--build-arg REPO_URL="$(REPO_URL)" \
	--build-arg LAST_COMMIT="$(LAST_COMMIT)" \
	--build-arg PROJECT_NAME="$(PROJECT_NAME)" \
	--build-arg VENDOR="$(VENDOR)" .
	@$(DOCKER) rmi -f $$(docker images -f "dangling=true" -q)

.PHONY: image.push
## image.push: Push the Docker image to the Github Registry
image.push:
	@$(DOCKER) push "$(DOCKER_REGISTRY)/$(PROJECT_NAME):$(VERSION)"

.PHONY: build.provider
build.provider:
	cd ./package && \
	rm -f *.xpkg && \
	pwd && \
	$(KUBECTL) crossplane build provider

.PHONY: push.provider
push.provider:
	cd ./package && \
	$(KUBECTL) crossplane push provider "$(DOCKER_REGISTRY)/crossplane-$(PROJECT_NAME):$(VERSION)"

.PHONY: ghcr.secret
ghcr.secret:
	$(KUBECTL) create secret docker-registry cr-token \
	--namespace crossplane-system --docker-server=ghcr.io \
	--docker-password=$(GITHUB_TOKEN) --docker-username=$(ORG_NAME)

.PHONY: docker.login
docker.login:
	docker login ghcr.io --username $(ORG_NAME) --password $(GITHUB_TOKEN)