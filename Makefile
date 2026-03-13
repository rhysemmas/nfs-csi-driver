# Build the CSI driver binary (local)
.PHONY: build
build:
	go build -o bin/csi-driver ./cmd/csi-driver

# Build Linux binary for container
.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/csi-driver ./cmd/csi-driver

# Build the Docker image
IMAGE_NAME ?= nfs-csi-driver
IMAGE_TAG  ?= latest

.PHONY: image
image:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

# Run tests
.PHONY: test
test:
	go test ./...

# Tidy modules
.PHONY: tidy
tidy:
	go mod tidy

# Install in cluster (after setting NFS server in deploy/config.yaml and deploy/controller-nfs-pv-pvc.yaml)
.PHONY: deploy
deploy:
	kubectl apply -f deploy/namespace.yaml
	kubectl apply -f deploy/config.yaml
	kubectl apply -f deploy/controller-nfs-pv-pvc.yaml
	kubectl apply -f deploy/rbac.yaml
	kubectl apply -f deploy/controller.yaml
	kubectl apply -f deploy/node.yaml
	kubectl apply -f deploy/storageclass.yaml

.PHONY: undeploy
undeploy:
	kubectl delete -f deploy/storageclass.yaml --ignore-not-found
	kubectl delete -f deploy/node.yaml --ignore-not-found
	kubectl delete -f deploy/controller.yaml --ignore-not-found
	kubectl delete -f deploy/rbac.yaml --ignore-not-found
	kubectl delete -f deploy/controller-nfs-pv-pvc.yaml --ignore-not-found
	kubectl delete -f deploy/config.yaml --ignore-not-found
	kubectl delete -f deploy/namespace.yaml --ignore-not-found
