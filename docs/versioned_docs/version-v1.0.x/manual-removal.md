---
title: Manual Removal
---

Create an `ImageList` and specify the images you would like to remove. In this case, the image `docker.io/library/alpine:3.7.3` will be removed.

```shell
cat <<EOF | kubectl apply -f -
apiVersion: eraser.sh/v1
kind: ImageList
metadata:
  name: imagelist
spec:
  images:
    - docker.io/library/alpine:3.7.3   # use "*" for all non-running images
EOF
```

> `ImageList` is a cluster-scoped resource and must be called imagelist. `"*"` can be specified to remove all non-running images instead of individual images.

Creating an `ImageList` should trigger an `ImageJob` that will deploy Eraser pods on every node to perform the removal given the list of images.

```shell
$ kubectl get pods -n eraser-system
eraser-system        eraser-controller-manager-55d54c4fb6-dcglq   1/1     Running   0          9m8s
eraser-system        eraser-kind-control-plane                    1/1     Running   0          11s
eraser-system        eraser-kind-worker                           1/1     Running   0          11s
eraser-system        eraser-kind-worker2                          1/1     Running   0          11s
```

Pods will run to completion and the images will be removed.

```shell
$ kubectl get pods -n eraser-system
eraser-system        eraser-controller-manager-6d6d5594d4-phl2q   1/1     Running     0          4m16s
eraser-system        eraser-kind-control-plane                    0/1     Completed   0          22s
eraser-system        eraser-kind-worker                           0/1     Completed   0          22s
eraser-system        eraser-kind-worker2                          0/1     Completed   0          22s
```

The `ImageList` custom resource status field will contain the status of the last job. The success and failure counts indicate the number of nodes the Eraser agent was run on.

```shell
$ kubectl describe ImageList imagelist
...
Status:
  Failed:     0
  Success:    3
  Timestamp:  2022-02-25T23:41:55Z
...
```

Verify the unused images are removed.

```shell
$ docker exec kind-worker ctr -n k8s.io images list | grep alpine
```

If the image has been successfully removed, there will be no output.
