# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= 0.0.1

SUDO=

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
# You can enable this value if you would like to use SHA Based Digests
# To enable set flag to true
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
	BUNDLE_GEN_FLAGS += --use-image-digests
endif

# Image URL to use all building/pushing image targets
IMG ?= ghcr.io/riotkit-org/backup-maker-controller:latest
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.24.2

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

.PHONY: coverage
coverage: test

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	@mkdir -p $(LOCALBIN)
	go build -o $(LOCALBIN)/manager main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

codegen-clientset:
	@echo "Generating Kubernetes Clients"
	./hack/update-codegen.sh
#	mkdir -p client
#	rm -rf client/clientset
#	mv ../client/clientset ./client/

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/.build
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
CONTROLLER_TOOLS_VERSION ?= v0.9.2

.PHONY: helm
helm:
	@for f in $$(ls ./config/crd/bases); do \
		echo -e "{{ if \$$.Values.installCRD }}\\n$$(cat ./config/crd/bases/$$f)\\n{{ end }}" > helm/templates/$$f; \
	done
	cp ./config/rbac/*_editor_role.yaml helm/templates/
	cp ./config/rbac/*_viewer_role.yaml helm/templates/
	role=$$(cat ./config/rbac/role.yaml); echo "$${role/'name: manager-role'/'name: backup-maker-controller-role'}" > helm/templates/controller-role.yaml

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

## testing

k3d:
	(${SUDO} docker ps | grep k3d-bm-server-0 > /dev/null 2>&1) || ${SUDO} k3d cluster create bm --registry-create bm-registry:0.0.0.0:5000
	${SUDO} k3d kubeconfig merge bm

k3d-dev: k3d skaffold-dev
skaffold-dev:
	cat /etc/hosts | grep "bm-registry" > /dev/null || (sudo /bin/bash -c "echo '127.0.0.1 bm-registry' >> /etc/hosts")

	export KUBECONFIG=~/.k3d/kubeconfig-bm.yaml
	kubectl apply -f config/crd/bases
	kubectl create ns backup-maker-controller || true
	skaffold dev -n backup-maker-controller --default-repo bm-registry:5000 --tag latest
