---
title: Quick Start
---

This tutorial demonstrates the functionality of Eraser and validates that non-running images are removed succesfully.

## Deploy a DaemonSet

After following the [install instructions](installation.md), we'll apply a demo `DaemonSet`. For illustrative purposes, a DaemonSet is applied and deleted so the non-running images remain on all nodes. The alpine image with the `3.7.3` tag will be used in this example. This is an image with a known critical vulnerability.

First, apply the `DaemonSet`:

```shell
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: alpine
spec:
  selector:
    matchLabels:
      app: alpine
  template:
    metadata:
      labels:
        app: alpine
    spec:
      containers:
      - name: alpine
        image: docker.io/library/alpine:3.7.3
EOF
```

Next, verify that the Pods are running or completed. After the `alpine` Pods complete, you may see a `CrashLoopBackoff` status. This is expected behavior from the `alpine` image and can be ignored for the tutorial.

```shell
$ kubectl get pods
NAME          READY   STATUS      RESTARTS     AGE
alpine-2gh9c   1/1     Running     1 (3s ago)   6s
alpine-hljp9   0/1     Completed   1 (3s ago)   6s
```

Delete the DaemonSet:

```shell
$ kubectl delete daemonset alpine
```

Verify that the Pods have been deleted:

```shell
$ kubectl get pods
No resources found in default namespace.
```

To verify that the `alpine` images are still on the nodes, exec into one of the worker nodes and list the images. If you are not using a kind cluster or Docker for your container nodes, you will need to adjust the exec command accordingly.

List the nodes:

```shell
$ kubectl get nodes
NAME                 STATUS   ROLES           AGE   VERSION
kind-control-plane   Ready    control-plane   45m   v1.24.0
kind-worker          Ready    <none>          45m   v1.24.0
kind-worker2         Ready    <none>          44m   v1.24.0
```

List the images then filter for `alpine`:

```shell
$ docker exec kind-worker ctr -n k8s.io images list | grep alpine
docker.io/library/alpine:3.7.3                                                                             application/vnd.docker.distribution.manifest.list.v2+json sha256:8421d9a84432575381bfabd248f1eb56f3aa21d9d7cd2511583c68c9b7511d10 2.0 MiB   linux/386,linux/amd64,linux/arm/v6,linux/arm64/v8,linux/ppc64le,linux/s390x  io.cri-containerd.image=managed
docker.io/library/alpine@sha256:8421d9a84432575381bfabd248f1eb56f3aa21d9d7cd2511583c68c9b7511d10           application/vnd.docker.distribution.manifest.list.v2+json sha256:8421d9a84432575381bfabd248f1eb56f3aa21d9d7cd2511583c68c9b7511d10 2.0 MiB   linux/386,linux/amd64,linux/arm/v6,linux/arm64/v8,linux/ppc64le,linux/s390x  io.cri-containerd.image=managed

```

## Automatically Cleaning Images

After deploying Eraser, it will automatically clean images in a regular interval. This interval can be set using the `manager.scheduling.repeatInterval` setting in the [configmap](https://eraser-dev.github.io/eraser/docs/customization#detailed-options). The default interval is 24 hours (`24h`). Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

Eraser will schedule collector pods to each node in the cluster, and each pod will contain 3 containers: collector, scanner, and eraser that will run to completion.

```shell
$ kubectl get pods -n eraser-system
NAMESPACE            NAME                                         READY   STATUS      RESTARTS         AGE
eraser-system        collector-kind-control-plane-sb789           0/3     Completed   0                26m
eraser-system        collector-kind-worker-j84hm                  0/3     Completed   0                26m
eraser-system        collector-kind-worker2-4lbdr                 0/3     Completed   0                26m
eraser-system        eraser-controller-manager-86cdb4cbf9-x8d7q   1/1     Running     0                26m
```

The collector container sends the list of all images to the scanner container, which scans and reports non-compliant images to the eraser container for removal of images that are non-running. Once all pods are completed, they will be automatically cleaned up. 

> If you want to remove all the images periodically, you can skip the scanner container by setting the `components.scanner.enabled` value to `false` using the [configmap](https://eraser-dev.github.io/eraser/docs/customization#detailed-options). In this case, each collector pod will hold 2 containers: collector and eraser.

```shell
$ kubectl get pods -n eraser-system
NAMESPACE            NAME                                         READY   STATUS      RESTARTS         AGE
eraser-system        collector-kind-control-plane-ksk2b           0/2     Completed   0                50s
eraser-system        collector-kind-worker-cpgqc                  0/2     Completed   0                50s
eraser-system        collector-kind-worker2-k25df                 0/2     Completed   0                50s
eraser-system        eraser-controller-manager-86cdb4cbf9-x8d7q   1/1     Running     0                55s
```
