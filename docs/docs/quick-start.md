---
title: Quick Start
sidebar_position: 3
---

This tutorial demonstrates the functionality of Eraser and validates that non-running images are removed after applying an `ImageList` with the declared images.

After following the [install instructions](https://example.com), we'll apply a demo `DaemonSet`. A `DaemonSet` is used so that when it is deleted, non-running images will remain on all nodes. We'll use the latest [Docker official hello-world image](https://hub.docker.com/_/hello-world)

First, apply the `DaemonSet`:

```shell
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: hello-world
spec:
  selector:
    matchLabels:
      app: hello-world
  template:
    metadata:
      labels:
        app: hello-world
    spec:
      containers:
      - name: hello-world
        image: hello-world:latest
EOF
```

Next, verify that the Pods are running or completed. After the `hello-world` Pods complete, you may see a `CrashLoopBackoff` status. This is expected behavior from the `hello-world` image and can be ignored for the tutorial.

```shell
kubectl get pods
```
```shell
NAME                READY   STATUS      RESTARTS     AGE
hello-world-2gh9c   1/1     Running     1 (3s ago)   6s
hello-world-hljp9   0/1     Completed   1 (3s ago)   6s
```

Delete the DaemonSet:

```shell
kubectl delete daemonset hello-world
```

Verify that the Pods have been deleted:

```shell
$ kubectl get pods
No resources found in default namespace.
```

