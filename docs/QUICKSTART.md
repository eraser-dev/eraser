- Install eraser from [deployment YAML](https://github.com/Azure/eraser/blob/v0.1.0/deploy/eraser.yaml).

```bash
$ kubectl apply -f https://raw.githubusercontent.com/Azure/eraser/v0.1.0/deploy/eraser.yaml
namespace/eraser-system created
customresourcedefinition.apiextensions.k8s.io/imagejobs.eraser.sh created
customresourcedefinition.apiextensions.k8s.io/imagelists.eraser.sh created
serviceaccount/eraser-controller-manager created
role.rbac.authorization.k8s.io/eraser-leader-election-role created
clusterrole.rbac.authorization.k8s.io/eraser-manager-role created
rolebinding.rbac.authorization.k8s.io/eraser-leader-election-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/eraser-manager-rolebinding created
configmap/eraser-manager-config created
deployment.apps/eraser-controller-manager created

$ kubectl get pods -n eraser-system
NAME                                         READY   STATUS    RESTARTS   AGE
eraser-controller-manager-759c7c8794-q8qqg   1/1     Running   0          22s
```

- Create an [ImageList](./test/e2e/test-data/eraser_v1alpha1_imagelist.yaml) and specify the images you would like to remove.

Example:
```bash
cat <<EOF | kubectl apply -f -
apiVersion: eraser.sh/v1alpha1
kind: ImageList
metadata:
  name: imagelist
spec:
  images:
    - sha256:2834dc507516af02784808c5f48b7cbe38b8ed5d0f4837f16e78d00deb7e7767
    - docker.io/library/nginx:latest
    - redis
EOF
```

> ImageList is a cluster-scoped resource and must be called `imagelist`. You can specify `"*"` for all non-running images.

- Creating an imagelist should trigger an [ImageJob](api/v1alpha1/imagejob_types.go) that will deploy [eraser](pkg/eraser/eraser.go) pods on every node to perform the removal given the list of images.

```bash
$ kubectl get pods -n eraser-system
eraser-system        eraser-controller-manager-55d54c4fb6-dcglq   1/1     Running   0          9m8s
eraser-system        eraser-kind-control-plane                    1/1     Running   0          11s
eraser-system        eraser-kind-worker                           1/1     Running   0          11s
eraser-system        eraser-kind-worker2                          1/1     Running   0          11s
```

- Pods will run to completion and the images will be removed.

```bash
$ kubectl get pods -n eraser-system
eraser-system        eraser-controller-manager-6d6d5594d4-phl2q   1/1     Running     0          4m16s
eraser-system        eraser-kind-control-plane                    0/1     Completed   0          22s
eraser-system        eraser-kind-worker                           0/1     Completed   0          22s
eraser-system        eraser-kind-worker2                          0/1     Completed   0          22s
```

- The `imagelist` custom resource status field will contain the status of the last job.

```bash
$ kubectl describe ImageList imagelist
...
Status:
  Failed:     0
  Success:    3
  Timestamp:  2022-02-25T23:41:55Z
...
```

By default, successful jobs will be deleted after a period of time. You can change this behavior by setting the following flags in the eraser-controller-manager:

- `--job-cleanup-on-success-delay`: Seconds to delay job deletion after successful runs. 0 means no delay. Defaults to `0`.
- `--job-cleanup-on-error-delay`: Seconds to delay job deletion after errored runs. 0 means no delay. Defaults to `86400` (24 hours).
- `--job-success-ratio`: Ratio of successful/total runs to consider a job successful. 1.0 means all runs must succeed. Defaults to `1.0`.