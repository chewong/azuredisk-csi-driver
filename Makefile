# Copyright 2017 The Kubernetes Authors.
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

PKG = sigs.k8s.io/azuredisk-csi-driver
GIT_COMMIT ?= $(shell git rev-parse HEAD)
REGISTRY ?= andyzhangx
DRIVER_NAME = disk.csi.azure.com
IMAGE_NAME = azuredisk-csi
IMAGE_VERSION ?= v0.7.0
# Use a custom version for E2E tests if we are in Prow
ifdef AZURE_CREDENTIALS
override IMAGE_VERSION := e2e-$(GIT_COMMIT)
endif
ifdef TEST_WINDOWS
IMAGE_VERSION = $(IMAGE_VERSION)-windows
endif
IMAGE_TAG = $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION)
IMAGE_TAG_LATEST = $(REGISTRY)/$(IMAGE_NAME):latest
REV = $(shell git describe --long --tags --dirty)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
TOPOLOGY_KEY = topology.$(DRIVER_NAME)/zone
ENABLE_TOPOLOGY ?= false
LDFLAGS ?= "-X ${PKG}/pkg/azuredisk.driverVersion=${IMAGE_VERSION} -X ${PKG}/pkg/azuredisk.gitCommit=${GIT_COMMIT} -X ${PKG}/pkg/azuredisk.buildDate=${BUILD_DATE} -X ${PKG}/pkg/azuredisk.DriverName=${DRIVER_NAME} -X ${PKG}/pkg/azuredisk.topologyKey=${TOPOLOGY_KEY} -extldflags "-static""
GINKGO_FLAGS = -ginkgo.noColor -ginkgo.v
ifeq ($(ENABLE_TOPOLOGY), true)
GINKGO_FLAGS += -ginkgo.focus="\[multi-az\]"
else
GINKGO_FLAGS += -ginkgo.focus="\[single-az\]"
endif
GOPATH ?= $(shell go env GOPATH)
GOBIN ?= $(GOPATH)/bin
GO111MODULE = off
export GOPATH GOBIN GO111MODULE

.PHONY: all
all: azuredisk

.PHONY: verify
verify:
	hack/verify-all.sh
	go vet ./pkg/...

.PHONY: unit-test
unit-test:
	go test -v -cover ./pkg/... ./test/utils/credentials

.PHONY: sanity-test
sanity-test: azuredisk
	go test -v -timeout=30m ./test/sanity

.PHONY: integration-test
integration-test: azuredisk
	go test -v -timeout=30m ./test/integration

.PHONY: e2e-test
e2e-test:
	go test -v -timeout=0 ./test/e2e ${GINKGO_FLAGS}

.PHONY: e2e-bootstrap
e2e-bootstrap: kustomize
	# Only build and push the image if it does not exist in the registry
	docker pull $(IMAGE_TAG) || make azuredisk-container push
	cd deploy && kustomize edit set image mcr.microsoft.com/k8s/csi/azuredisk-csi=$(IMAGE_TAG)
	kustomize build deploy | kubectl apply -f -

.PHONY: e2e-bootstrap-windows
e2e-bootstrap-windows: kustomize
	# Only build and push the image if it does not exist in the registry
	docker pull $(IMAGE_TAG) || make azuredisk-container-windows push
	cd deploy/windows && kustomize edit set image mcr.microsoft.com/k8s/csi/azuredisk-csi=$(IMAGE_TAG)
	kustomize build deploy/windows | kubectl apply -f -

.PHONY: e2e-teardown
e2e-teardown:
	kustomize build deploy | kubectl delete -f -

.PHONY: e2e-teardown-windows
e2e-teardown-windows:
	kustomize build deploy/windows | kubectl delete -f -

.PHONY: azuredisk
azuredisk:
	if [ ! -d ./vendor ]; then dep ensure -vendor-only; fi
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags ${LDFLAGS} -o _output/azurediskplugin ./pkg/azurediskplugin

.PHONY: azuredisk-windows
azuredisk-windows:
	if [ ! -d ./vendor ]; then dep ensure -vendor-only; fi
	CGO_ENABLED=0 GOOS=windows go build -a -ldflags ${LDFLAGS} -o _output/azurediskplugin.exe ./pkg/azurediskplugin

.PHONY: container
container: azuredisk
	docker build --no-cache -t $(IMAGE_TAG) -f ./pkg/azurediskplugin/Dockerfile .

.PHONY: azuredisk-container
azuredisk-container: azuredisk
	docker build --no-cache -t $(IMAGE_TAG) -f ./pkg/azurediskplugin/Dockerfile .

.PHONY: azuredisk-container-windows
azuredisk-container-windows: azuredisk-windows
	docker build --no-cache --platform windows/amd64 -t $(IMAGE_TAG) -f ./pkg/azurediskplugin/Windows.Dockerfile .

.PHONY: push
push:
	docker push $(IMAGE_TAG)

.PHONY: push-latest
push-latest:
	docker tag $(IMAGE_TAG) $(IMAGE_TAG_LATEST)
	docker push $(IMAGE_TAG_LATEST)

.PHONY: build-push
build-push: azuredisk-container
	docker tag $(IMAGE_TAG) $(IMAGE_TAG_LATEST)
	docker push $(IMAGE_TAG_LATEST)

.PHONY: clean
clean:
	go clean -r -x
	-rm -rf _output

.PHONY: kustomize
kustomize:
	GO111MODULE=on go get sigs.k8s.io/kustomize/kustomize/v3@v3.3.0
