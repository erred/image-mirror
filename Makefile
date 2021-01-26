UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
	OS = linux
endif
ifeq ($(UNAME_S),Darwin)
	OS = darwin
endif

ARCH ?= amd64

IMG ?= docker.io/seankhliao/image-mirror
VERSION ?= latest
CLUSTER_NAME ?= image-mirror

KIND_VERSION 		?= v0.10.0
KUBECTL_VERSION 	?= v0.20.2
KUSTOMIZE_VERSION 	?= v3.9.2
SKAFFOLD_VERSION 	?= v1.18.0

KIND 		?= $$(pwd)/bin/kind
KUBECTL 	?= $$(pwd)/bin/kubectl
KUSTOMIZE 	?= $$(pwd)/bin/kustomize
SKAFFOLD 	?= $$(pwd)/bin/skaffold

.PHONY: clean_cluster
clean_cluster:
	$(KIND) delete cluster --name $(CLUSTER_NAME) || true

.PHONY: tools
tools:
	curl --create-dirs -Lo $(KIND) https://github.com/kubernetes-sigs/kind/releases/download/$(KIND_VERSION)/kind-$(OS)-$(ARCH) && chmod +x $(KIND)
	curl --create-dirs -Lo $(KUBECTL) https://dl.k8s.io/release/$(KUBECTL_VERSION)/bin/$(OS)/$(ARCH)/kubectl && chmod +x $(KUBECTL)
	curl --create-dirs -L https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2F$(KUSTOMIZE_VERSION)/kustomize_$(KUSTOMIZE_VERSION)_$(OS)_$(ARCH).tar.gz | tar xz -C $$(pwd)/bin && chmod +x $(KUSTOMIZE)
	curl --create-dirs -Lo $(SKAFFOLD) https://storage.googleapis.com/skaffold/releases/$(SKAFFOLD_VERSION)/skaffold-$(OS)-$(ARCH) && chmod +x $(SKAFFOLD)

.PHONY: cluster
cluster:
	$(KIND) create cluster --name $(CLUSTER_NAME)

.PHONY: dev
dev:
	$(SKAFFOLD) dev
