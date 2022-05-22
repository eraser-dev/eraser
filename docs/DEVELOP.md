
# Developer Guide

Developing this project requires access to a Kubernetes cluster and Go version 1.16 or later.

We hope this help ease iterative local development.

## Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/)
- [docker](https://docs.docker.com/get-docker/)
- [Tilt](https://docs.tilt.dev/install.html)

## Steps

1. Creating a local cluster using kind.
     - Follow instructions on `kind` website for install.
     - kind should now be installed and ready for use.
     - `./kindLocalRegistry.sh` should have you up and running for local development, this is a direct copy from the kind documentation: <https://kind.sigs.k8s.io/docs/user/local-registry/> _Kind switches `kubectl` context to `kind-kind` so you can work with your LOCAL kind cluster using kubectl_
       - this script instantiates a kind cluster with local repository linked and ready for local development.

    A successful cluster creation should produce the below output.

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

2. Installing tilt
   - `curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash`
   - once installed run the command `tilt up`, there is a `./Tiltfile` provided, please review and update as you work
     - within the tilt file we are building a local image and using the release yaml to deploy to the local kind cluster.
    A successful tilt up will have you on a screen that looks like below

    ```bash
    Tilt started on http://localhost:10350/
    v0.30.0, built 2022-05-12

    (space) to open the browser
    (s) to stream logs (--stream=true)
    (t) to open legacy terminal mode (--legacy=true)
    (ctrl-c) to exit
    ```

   - Hit the space bar for the browser view.
