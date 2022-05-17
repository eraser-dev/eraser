# Quickstart Tutorial

1. Clone the Eraser repo
    ```shell
    git clone https://github.com/Azure/eraser.git
    cd eraser
    ```
2. Create a cluster

    This will install and create a Kind cluster with a control-plane and two worker nodes.

    ``` shell
    curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.13.0/kind-linux-amd64
    chmod +x ./kind
    sudo mv ./kind /usr/bin
    kind create cluster --config test/e2e/kind-config.yaml
    ```
    
    Verify your cluster is running.
  
    ```shell
    $ kubectl cluster-info --context kind-kind
    Kubernetes control plane is running at https://127.0.0.1:40989
    CoreDNS is running at https://127.0.0.1:40989/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy

    To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.
    ``` 


3. Deploy Eraser to the Cluster

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

4. Apply and delete a deployment

    Applying and deleting a deployment will leave unused images on the nodes where it was deployed. For this example, an `ngnix` deployment will be used. In following steps, these images will be used to verify that Eraser is removing the correct images.

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
    spec:
      selector:
        matchLabels:
          app: nginx
      replicas: 2 
      template:
        metadata:
          labels:
            app: nginx
        spec:
          containers:
          - name: nginx
            image: nginx:1.14.2
            ports:
            - containerPort: 80
    EOF
    ```

    ```shell
    # list pods in the default namespace
    $ kubectl get po
    NAME                                READY   STATUS    RESTARTS   AGE
    nginx-deployment-6595874d85-bvmg4   1/1     Running   0          8s
    nginx-deployment-6595874d85-h2r9c   1/1     Running   0          7s
    ```

    ```shell
    $ kubectl delete deployment nginx-deployment
    deployment.apps "nginx-deployment" deleted
    ```

    ```shell
    # verify the pods have been deleted
    $ kubectl get po
    No resources found in default namespace.
    ```

5. List images on a worker node

    To verify that the `nginx` images are still on the nodes, exec into one of the worker nodes and list the images.

  ```shell
  # get a list of the nodes
  $ kubectl get no
  NAME                 STATUS   ROLES           AGE   VERSION
  kind-control-plane   Ready    control-plane   45m   v1.24.0
  kind-worker          Ready    <none>          45m   v1.24.0
  kind-worker2         Ready    <none>          44m   v1.24.0
  ```

  ``` shell
  # list the images then filter for nginx
  $ docker exec kind-worker ctr -n k8s.io images list | grep nginx
  docker.io/library/nginx:1.14.2                                                                  application/vnd.docker.distribution.manifest.list.v2+json sha256:f7988fb6c02e0ce69257d9bd9cf37ae20a60f1df7563c3a2a6abe24160306b8d 42.6 MiB  linux/386,linux/amd64,linux/arm/v7,linux/arm64/v8,linux/ppc64le,linux/s390x  io.cri-containerd.image=managed 
  docker.io/library/nginx@sha256:f7988fb6c02e0ce69257d9bd9cf37ae20a60f1df7563c3a2a6abe24160306b8d application/vnd.docker.distribution.manifest.list.v2+json sha256:f7988fb6c02e0ce69257d9bd9cf37ae20a60f1df7563c3a2a6abe24160306b8d 42.6 MiB  linux/386,linux/amd64,linux/arm/v7,linux/arm64/v8,linux/ppc64le,linux/s390x  io.cri-containerd.image=managed 
  ```

6. Create an ImageList
    Create an [ImageList](../test/e2e/test-data/eraser_v1alpha1_imagelist.yaml) and specify the images you would like to remove. In this case, the image `docker.io/library/nginx:1.14.2` will be removed.

      ```bash
      cat <<EOF | kubectl apply -f -
      apiVersion: eraser.sh/v1alpha1
      kind: ImageList
      metadata:
        name: imagelist
      spec:
        images:
          - docker.io/library/nginx:1.14.2
      EOF
      ```


> ImageList is a cluster-scoped resource and must be called `imagelist`. You can specify `"*"` for all non-running images.

  Creating an imagelist should trigger an [ImageJob](api/v1alpha1/imagejob_types.go) that will deploy [eraser](../pkg/eraser/eraser.go) pods on every node to perform the removal given the list of images.

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


7. Verify the unused images are removed

    ``` shell
    docker exec <nodeName> ctr -n k8s.io images list | grep nginx
    ```

    If the image has been successfully removed, there will be no output. 

8. Tear down your cluster

    ```shell
    kind delete cluster kind
    ```