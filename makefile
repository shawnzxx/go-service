# Check to see if we can use ash, in Alpine images, or default to BASH.
SHELL_PATH = /bin/ash
SHELL = $(if $(wildcard $(SHELL_PATH)),/bin/ash,/bin/bash)

# ==============================================================================
# CLASS NOTES
#
# Kind
# 	For full Kind v0.18 release notes: https://github.com/kubernetes-sigs/kind/releases/tag/v0.18.0
#
# You can use openSSL to test generate RSA Keys pair
# 	To generate a private/public key PEM file.
# 	$ openssl genpkey -algorithm RSA -out private.pem -pkeyopt rsa_keygen_bits:2048
# 	$ openssl rsa -pubout -in private.pem -out public.pem

#
# OPA Playground
# 	https://play.openpolicyagent.org/
# 	https://academy.styra.com/
# 	https://www.openpolicyagent.org/docs/latest/policy-reference/

# ==============================================================================
# Brew Installation
#
#	Having brew installed will simplify the process of installing all the tooling.
#
#	Run this command to install brew on your machine. This works for Linux, Mac and Windows.
#	The script explains what it will do and then pauses before it does it.
#	$ /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
#
# 	Install GCC:
#	$ brew install gcc

# ==============================================================================
# Install Tooling and Dependencies
#
#   This project uses Docker and it is expected to be installed. Please provide
#   Docker at least 3 CPUs.
#
#	Run these commands to install everything needed.
#	$ make dev-brew
#	$ make dev-docker
#	$ make dev-gotooling

# ==============================================================================
# CLASS NOTES
#
# Kind
# 	For full Kind v0.20 release notes: https://github.com/kubernetes-sigs/kind/releases/tag/v0.20.0

# ==============================================================================
# Define dependencies

GOLANG          := golang:1.21.3
ALPINE          := alpine:3.18
KIND            := kindest/node:v1.27.3
POSTGRES        := postgres:15.4
TELEPRESENCE    := datawire/tel2:2.16.1

KIND_CLUSTER    := ardan-starter-cluster
NAMESPACE       := sales-system
APP             := sales
BASE_IMAGE_NAME := ardanlabs/service
SERVICE_NAME    := sales-api
VERSION         := 0.0.1
SERVICE_IMAGE   := $(BASE_IMAGE_NAME)/$(SERVICE_NAME):$(VERSION)

# VERSION       := "0.0.1-$(shell git rev-parse --short HEAD)"

# ==============================================================================
# Install dependencies

dev-gotooling:
	go install github.com/divan/expvarmon@latest
	go install github.com/rakyll/hey@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install golang.org/x/tools/cmd/goimports@latest

dev-brew:
	brew update
	brew tap hashicorp/tap
	brew list kind || brew install kind
	brew list kubectl || brew install kubectl
	brew list kustomize || brew install kustomize
	brew list pgcli || brew install pgcli
	brew list vault || brew install vault
	brew install helm


dev-docker:
	docker pull $(GOLANG)
	docker pull $(ALPINE)
	docker pull $(KIND)
	docker pull $(POSTGRES)
	docker pull $(TELEPRESENCE)
	
# ==============================================================================
# Building containers
build:
	docker build \
		-f zarf/docker/dockerfile.service \
		-t $(SERVICE_IMAGE) \
		--build-arg BUILD_REF=$(VERSION) \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		.

# Docker remove <none> TAG images:
# https://stackoverflow.com/questions/33913020/docker-remove-none-tag-images
remove-none-images:
	docker rmi $(docker images --filter "dangling=true" -q --no-trunc)

# ==============================================================================
# bootstrap the dev cluster

# dev cluster all in one up
#telepresence password is @SHxx20xxxx0x04
dev-up: dev-up-local
	telepresence --context=kind-$(KIND_CLUSTER) helm install
	telepresence --context=kind-ardan-starter-cluster quit -u
	telepresence --context=kind-$(KIND_CLUSTER) connect
# dev cluster all in one down
dev-down:
	kind delete cluster --name $(KIND_CLUSTER)

# if you have issue to run telepresence, run it step by step
# follow this link: https://www.telepresence.io/docs/latest/quick-start/
# step 1
dev-up-local:
	kind create cluster \
		--image $(KIND) \
		--name $(KIND_CLUSTER) \
		--config zarf/k8s/dev/kind-config.yaml
# what is local-path-storage namespace: https://mauilion.dev/posts/kind-pvc/
	kubectl wait --timeout=120s --namespace=local-path-storage --for=condition=Available deployment/local-path-provisioner
	
	kind load docker-image $(TELEPRESENCE) --name $(KIND_CLUSTER)
	kind load docker-image $(POSTGRES) --name $(KIND_CLUSTER)


# step 2
# if need password is @SHxx20xxxx0x04
dev-load-telepresence:
	kind load docker-image $(TELEPRESENCE) --name $(KIND_CLUSTER)
	telepresence --context=kind-ardan-starter-cluster helm install
	telepresence --context=kind-ardan-starter-cluster quit -u
	telepresence --context=kind-ardan-starter-cluster connect

# ==============================================================================
# re=deploy service on cluster

# if you changed the code then run this command to re-build the service
dev-update: build dev-load dev-restart

# if you changed the k8s configuration then run this command to re-apply new settings
dev-update-apply: build dev-load dev-apply

dev-load:
	kind load docker-image $(SERVICE_IMAGE) --name $(KIND_CLUSTER)

dev-restart:
	kubectl rollout restart deployment $(APP) --namespace=$(NAMESPACE)

dev-apply:
	kustomize build zarf/k8s/dev/database | kubectl apply -f -
	kubectl rollout status --namespace=$(NAMESPACE) --watch --timeout=120s sts/database

	kustomize build zarf/k8s/dev/sales | kubectl apply -f -
	kubectl wait pods --namespace=$(NAMESPACE) --selector app=$(APP) --for=condition=Ready

# ------------------------------------------------------------------------------
# run monitoring commands

# check dev status
dev-status:
	kubectl get nodes -o wide
	kubectl get svc -o wide
	kubectl get pods -o wide --watch --all-namespaces

# check dev logs
dev-logs:
	kubectl logs --namespace=$(NAMESPACE) -l app=$(APP) --all-containers=true -f --tail=100 | go run app/tooling/logfmt/main.go -service=$(SERVICE_NAME)
	
dev-describe-deployment:
	kubectl describe deployment --namespace=$(NAMESPACE) $(APP)

dev-describe-sales:
	kubectl describe pod --namespace=$(NAMESPACE) -l app=$(APP)

# ==============================================================================
# run commands
run-scratch:
	go run app/tooling/scratch/main.go

run-local:
	go run app/services/sales-api/main.go

run-local-help:
	go run app/services/sales-api/main.go --help

tidy:
	go mod tidy
	go mod vendor

metrics-view:
	expvarmon -ports="$(SERVICE_NAME).$(NAMESPACE).svc.cluster.local:4000" -vars="build,requests,goroutines,errors,panics,mem:memstats.Alloc"

metrics-view-local:
	expvarmon -ports="localhost:4000" -vars="build,requests,goroutines,errors,panics,mem:memstats.Alloc"

test-endpoint:
# k8s DNS location: https://yuminlee2.medium.com/kubernetes-dns-bdca7b7cb868#:~:text=In%20Kubernetes%2C%20DNS%20names%20are%20assigned%20to%20Pods%20and%20Services,format%20.
	curl -il $(SERVICE_NAME).$(NAMESPACE).svc.cluster.local:3000/test

test-endpoint-local:
	curl -il localhost:3000/test

# before tun commands below, you need to pump in token
# `make run-scratch`, copy paste token value from the output
# write token value into env variable: `export TOKEN=$token`
# then run commands below
test-endpoint-auth:
	curl -il -H "Authorization: Bearer ${TOKEN}" $(SERVICE_NAME).$(NAMESPACE).svc.cluster.local:3000/test/auth

test-endpoint-auth-local:
	curl -il -H "Authorization: Bearer ${TOKEN}" localhost:3000/test/auth

liveness-local:
	curl -il http://localhost:4000/debug/liveness

liveness:
	curl -il http://$(SERVICE_NAME).$(NAMESPACE).svc.cluster.local:4000/debug/liveness

readiness-local:
	curl -il http://localhost:4000/debug/readiness

readiness:
	curl -il http://$(SERVICE_NAME).$(NAMESPACE).svc.cluster.local:4000/debug/readiness

pgcli-local:
	pgcli postgresql://postgres:postgres@localhost

pgcli:
	pgcli postgresql://postgres:postgres@database-service.$(NAMESPACE).svc.cluster.local

migrate:
	go run app/tooling/admin/main.go
