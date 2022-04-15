# Set the shell to bash always
SHELL := /bin/bash

# Look for a .env file, and if present, set make variables from it.
ifneq (,$(wildcard ./.env))
	include ./.env
	export $(shell sed 's/=.*//' ./.env)
endif

KIND_CLUSTER_NAME ?= local-dev
KUBECONFIG ?= $(HOME)/.kube/config

PROJECT_NAME := provider-git
ORG_NAME := krateoplatformops

VERSION := $(shell git describe --dirty --always --tags | sed 's/-/./2' | sed 's/-/./2')
ifndef VERSION
VERSION := v0.0.0
endif

# Github Container Registry
DOCKER_REGISTRY := ghcr.io/$(ORG_NAME)

# Tools
KIND=$(shell which kind)
KUBECTL=$(shell which kubectl)


.DEFAULT_GOAL := help

.PHONY: kind.up
kind.up: ## Starts a KinD cluster for local development
	@$(KIND) get kubeconfig --name $(KIND_CLUSTER_NAME) >/dev/null 2>&1 || $(KIND) create cluster --name=$(KIND_CLUSTER_NAME)


.PHONY: install.crossplane
install.crossplane: ## Install Crossplane into the local KinD cluster
	$(KUBECTL) create namespace crossplane-system || true
	helm repo add crossplane-stable https://charts.crossplane.io/stable
	helm repo update
	helm install crossplane --namespace crossplane-system crossplane-stable/crossplane


.PHONY: install.provider
install.provider: ## Install this provider
	@$(KUBECTL) apply -f ./provider.yaml

.PHONY: kind.down
kind.down: ## Shuts down the KinD cluster
	@$(KIND) delete cluster --name=$(KIND_CLUSTER_NAME)


.PHONY: cr.token
cr.token: ## Create the secret for container registry credentials
	@$(KUBECTL) create secret docker-registry cr-token \
	--namespace crossplane-system --docker-server=ghcr.io \
	--docker-password=$(GITHUB_TOKEN) --docker-username=$(ORG_NAME)


.PHONY: help
help: ## Print this help
	@grep -E '^[a-zA-Z\._-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
