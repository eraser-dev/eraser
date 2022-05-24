# Quickstart Tutorial

This tutorial demonstrates the functionality of Eraser and validates that non-running images are removed after applying an `ImageList` with the declared images.

## Prerequisites:
- [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)

## Steps

1. Create a cluster

    Eraser can be deployed to any Kubernetes cluster, however, for the purposes of this tutorial a [kind](https://kind.sigs.k8s.io/) cluster will be used.
  
    For information on how to install `kind` see the [kind quick start guide](https://kind.sigs.k8s.io/docs/user/quick-start/#installation). Kind also requires that you have [Docker](https://docs.docker.com/get-docker/) installed and configured.

    After installing `kind`, paste the following in your terminal to create a cluster with a control-plane and two worker nodes.

    ```bash
    cat <<EOF | kind create cluster --config -
    kind: Cluster
    apiVersion: kind.x-k8s.io/v1alpha4
    nodes:
    - role: control-plane
    - role: worker
    - role: worker
    EOF
    ``` 

    A successful cluster creation should produce an output similar to the one below.

    ```bash
    Creating cluster "kind" ...
    ✓ Ensuring node image (kindest/node:v1.24.0) 🖼
    ✓ Preparing nodes 📦 📦 📦  
    ✓ Writing configuration 📜 
    ✓ Starting control-plane 🕹️ 
    ✓ Installing CNI 🔌 
    ✓ Installing StorageClass 💾 
    ✓ Joining worker nodes 🚜 
    Set kubectl context to "kind-kind"
    You can now use your cluster with:

    kubectl cluster-info --context kind-kind
    ```


2. Deploy Eraser to the Cluster

    Eraser can be deployed by applying the mainifest file at `https://raw.githubusercontent.com/Azure/eraser/v0.1.0/deploy/eraser.yaml`. If you prefer using [Helm](https://helm.sh/) for your deployments, see the [Eraser Helm Chart README](../manifest_staging/charts/eraser/README.md) for usage.

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
    ```

    ```bash
    $ kubectl get pods -n eraser-system
    NAME                                         READY   STATUS    RESTARTS   AGE
    eraser-controller-manager-759c7c8794-q8qqg   1/1     Running   0          22s
    ```

3. Apply and delete a DaemonSet

    For illustrative purposes, a DaemonSet is applied and deleted so the non-running images remain on all nodes. The [Docker official hello-world image](https://hub.docker.com/_/hello-world) with the `latest` tag will be used in this example. However, it is not recommended to use `latest` in a production environment and is only used to demonstrate the Eraser functionality.

    In the following steps, these images will be used to verify that Eraser is removing the correct images.

    ```bash
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

    Verify the pods are running or completed. After the `hello-world` pods complete, you may see a `CrashLoopBackOff` status. This is expected behavior from the `hello-world` image and can be ignored for the purposes of this tutorial.

    ```bash
    $ kubectl get pods
    NAME                READY   STATUS      RESTARTS     AGE
    hello-world-2gh9c   1/1     Running     1 (3s ago)   6s
    hello-world-hljp9   0/1     Completed   1 (3s ago)   6s
    ```

    Delete the DaemonSet:

    ```bash
    $ kubectl delete daemonset hello-world
    daemonset.apps "hello-world" deleted
    ```

    Verify the pods have been deleted:

    ```bash
    $ kubectl get pods
    No resources found in default namespace.
    ```

4. List images on a worker node

    To verify that the `hello-world` images are still on the nodes, exec into one of the worker nodes and list the images. If you are not using a `kind` cluster or `Docker` for your container nodes, you will need to adjust the exec command accordingly. 

    List the nodes:
    ```bash
    $ kubectl get nodes
    NAME                 STATUS   ROLES           AGE   VERSION
    kind-control-plane   Ready    control-plane   45m   v1.24.0
    kind-worker          Ready    <none>          45m   v1.24.0
    kind-worker2         Ready    <none>          44m   v1.24.0
    ```

    List the images then filter for hello-world:

    ``` bash
    $ docker exec kind-worker ctr -n k8s.io images list | grep hello-world
    docker.io/library/hello-world:latest                                                                  application/vnd.docker.distribution.manifest.list.v2+json sha256:80f31da1ac7b312ba29d65080fddf797dd76acfb870e677f390d5acba9741b17 6.9 KiB   linux/386,linux/amd64,linux/arm/v5,linux/arm/v7,linux/arm64/v8,linux/mips64le,linux/ppc64le,linux/riscv64,linux/s390x,windows/amd64 io.cri-containerd.image=managed 
    docker.io/library/hello-world@sha256:80f31da1ac7b312ba29d65080fddf797dd76acfb870e677f390d5acba9741b17 application/vnd.docker.distribution.manifest.list.v2+json sha256:80f31da1ac7b312ba29d65080fddf797dd76acfb870e677f390d5acba9741b17 6.9 KiB   linux/386,linux/amd64,linux/arm/v5,linux/arm/v7,linux/arm64/v8,linux/mips64le,linux/ppc64le,linux/riscv64,linux/s390x,windows/amd64 io.cri-containerd.image=managed 
    ```

5. Create an ImageList

    Create an [ImageList](../test/e2e/test-data/eraser_v1alpha1_imagelist.yaml) and specify the images you would like to remove. In this case, the image `docker.io/library/hello-world:latest` will be removed.

      ```bash
      cat <<EOF | kubectl apply -f -
      apiVersion: eraser.sh/v1alpha1
      kind: ImageList
      metadata:
        name: imagelist
      spec:
        images:
          - docker.io/library/hello-world:latest
      EOF
      ```

    > ImageList is a cluster-scoped resource and must be called `imagelist`. You can specify `"*"` for all non-running images.

    Creating an `imagelist` should trigger an [ImageJob](api/v1alpha1/imagejob_types.go) that will deploy Eraser pods on every node to perform the removal given the list of images.

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

    The `imagelist` custom resource status field will contain the status of the last job. The success and failure counts indicate the number of nodes the Eraser agent was run on.

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

6. Verify the unused images are removed

    ``` bash
    docker exec kind-worker ctr -n k8s.io images list | grep hello-world
    ```

    If the image has been successfully removed, there will be no output. 

7. Tear down your cluster

    ```bash
    kind delete cluster
    ```