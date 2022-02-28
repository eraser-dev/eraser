# Eraser: Cleaning up Images from Kubernetes Nodes

Eraser helps Kubernetes admins remove a list of non-running images from all Kubernetes nodes in a cluster.

> 🚨 This project is currently in early alpha stage. Do NOT use for production.

## Getting started

Create an [ImageList](./test/e2e/test-data/eraser_v1alpha1_imagelist.yaml) and specify the images you would like to remove.

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

ImageList is a cluster-scoped resource and must be called `imagelist`. You can specify `"*"` for all non-running images.

Creating an imagelist should trigger an [ImageJob](api/v1alpha1/imagejob_types.go) that will deploy [eraser](pkg/eraser/eraser.go) pods on every node to perform the removal given the list of images.

```bash
$ kubectl get pods -n eraser-system
eraser-system        eraser-controller-manager-55d54c4fb6-dcglq   1/1     Running   0          9m8s
eraser-system        eraser-kind-control-plane                    1/1     Running   0          11s
eraser-system        eraser-kind-worker                           1/1     Running   0          11s
eraser-system        eraser-kind-worker2                          1/1     Running   0          11s
```

Pods will run to completion and the images will be removed.

```bash
$ kubectl get pods -n eraser-system
eraser-system        eraser-controller-manager-6d6d5594d4-phl2q   1/1     Running     0          4m16s
eraser-system        eraser-kind-control-plane                    0/1     Completed   0          22s
eraser-system        eraser-kind-worker                           0/1     Completed   0          22s
eraser-system        eraser-kind-worker2                          0/1     Completed   0          22s
```

The `imagelist` custom resource status field will contain the status of the last job.

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

## Developer Setup

Developing this project requires access to a Kubernetes cluster and Go version 1.16 or later.

## Design and Future Work
* [Design Documentation](https://docs.google.com/document/d/1Rz1bkZKZSLVMjC_w8WLASPDUjfU80tjV-XWUXZ8vq3I/edit?usp=sharing)

## Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us the rights to use your contribution. For details, visit https://cla.opensource.microsoft.com.

When you submit a pull request, a CLA bot will automatically determine whether you need to provide a CLA and decorate the PR appropriately (e.g., status check, comment). Simply follow the instructions provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/). For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

## Trademarks

This project may contain trademarks or logos for projects, products, or services. Authorized use of Microsoft trademarks or logos is subject to and must follow [Microsoft's Trademark & Brand Guidelines](https://www.microsoft.com/en-us/legal/intellectualproperty/trademarks/usage/general). Use of Microsoft trademarks or logos in modified versions of this project must not cause confusion or imply Microsoft sponsorship. Any use of third-party trademarks or logos are subject to those third-party's policies.
