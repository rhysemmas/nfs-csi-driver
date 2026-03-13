# NFS CSI Driver for Kubernetes

A [Container Storage Interface (CSI)](https://kubernetes-csi.github.io/docs/) driver that provisions Kubernetes volumes as **directories on an NFS server**. Each `PersistentVolumeClaim` gets a dedicated directory on the NFS export; pods mount that directory via NFS.

## How it works

1. **Controller** (single replica): Mounts the NFS server’s export (e.g. `/exports`) at a local path. On `CreateVolume` it creates a new subdirectory (one per volume); on `DeleteVolume` it removes that directory. It runs alongside the [external-provisioner](https://kubernetes-csi.github.io/docs/external-provisioner.html) sidecar, which watches `PersistentVolumeClaim` objects and calls the CSI controller.
2. **Node** (DaemonSet): On each node, when a pod uses a PVC, the node plugin runs `NodePublishVolume`: it mounts the NFS share (`server:path`) to the pod’s volume path. Unpublish unmounts it.

So: **one NFS export** (e.g. `nfs-server:/exports`) → **one directory per PVC** (e.g. `/exports/pvc-<uid>`) → **NFS mount in the pod**.

## Prerequisites

- A running **NFS server** that exports a path (e.g. `/exports`). All volume directories will be created under that path.
- Kubernetes cluster with CSI support (1.20+).
- Nodes must have **NFS client** utilities (`mount.nfs`); the driver image is based on Alpine and includes them.

## Build

### Local binary

```bash
# Install dependencies and build
go mod tidy
go build -o bin/csi-driver ./cmd/csi-driver
```

Or:

```bash
make build
```

### Docker image (for in-cluster deployment)

```bash
docker build -t nfs-csi-driver:latest .
# Or with Makefile:
make image
# Custom name/tag:
make image IMAGE_NAME=myregistry/nfs-csi IMAGE_TAG=v0.1.0
```

To run the controller and node pods, push the image to a registry your cluster can pull from, or load it on a kind/minikube cluster:

```bash
# kind example
kind load docker-image nfs-csi-driver:latest
```

## Install in a Kubernetes cluster

### 1. Configure the NFS server

Edit `deploy/config.yaml` and set your NFS server and export path:

```yaml
data:
  NFS_SERVER: "nfs-server.default.svc.cluster.local"   # or IP, e.g. 10.0.0.5
  NFS_ROOT_PATH: "/exports"
```

Edit `deploy/controller.yaml` so the NFS volume matches the same server and path:

```yaml
volumes:
  - name: nfs-root
    nfs:
      server: nfs-server.default.svc.cluster.local
      path: /exports
```

### 2. Use your built image

In both `deploy/controller.yaml` and `deploy/node.yaml`, set the CSI driver image to the one you built:

```yaml
containers:
  - name: csi-driver
    image: nfs-csi-driver:latest   # or your registry/image:tag
```

### 3. Deploy the driver

```bash
kubectl apply -f deploy/namespace.yaml
kubectl apply -f deploy/config.yaml
kubectl apply -f deploy/rbac.yaml
kubectl apply -f deploy/controller.yaml
kubectl apply -f deploy/node.yaml
kubectl apply -f deploy/storageclass.yaml
```

Or:

```bash
make deploy
```

### 4. Verify

- Controller and node pods should be running:

```bash
kubectl get pods -n nfs-csi
kubectl get ds -n nfs-csi
```

- Create a test PVC and pod:

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: test-pvc
spec:
  accessModes: [ReadWriteMany]
  storageClassName: nfs-csi
  resources:
    requests:
      storage: 1Gi
---
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  containers:
    - name: app
      image: busybox
      command: ["sleep", "infinity"]
      volumeMounts:
        - name: data
          mountPath: /data
  volumes:
    - name: data
      persistentVolumeClaim:
        claimName: test-pvc
EOF
```

- Check that a new directory appeared on the NFS server under the export path and that the pod can use the volume (e.g. `kubectl exec test-pod -- touch /data/hello`).

## Driver options (flags)

| Flag | Description | Controller | Node |
|------|-------------|------------|------|
| `--endpoint` | CSI gRPC endpoint (e.g. `unix:///csi/csi.sock`) | ✓ | ✓ |
| `--driver-name` | Driver name (default `nfs.csi.nootnoot.co.uk`) | ✓ | ✓ |
| `--node-id` | Kubernetes node name (required for node) | — | ✓ |
| `--nfs-server` | NFS server hostname/IP | ✓ | — |
| `--nfs-root-path` | Exported path on NFS server | ✓ | — |
| `--nfs-root-mount` | Local path where NFS is mounted in controller (default `/nfs-root`) | ✓ | — |

## Uninstall

```bash
# Delete StorageClass and driver workloads
kubectl delete -f deploy/storageclass.yaml
kubectl delete -f deploy/node.yaml
kubectl delete -f deploy/controller.yaml
kubectl delete -f deploy/rbac.yaml
kubectl delete -f deploy/config.yaml
kubectl delete -f deploy/namespace.yaml
```

Or:

```bash
make undeploy
```

## License

See repository license.
# nfs-csi-driver
