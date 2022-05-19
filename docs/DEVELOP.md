Developing this project requires access to a Kubernetes cluster and Go version 1.16 or later.

### [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/)

- working within a `codespace` you can install and instantiate a kind cluster for testing
  - Follow instructions on `kind` website for install from binary.
  - Move kind binary to `/usr/bin`:`mv ./kind /usr/bin`
  - kind should now be installed and ready
  - `kind create cluster` should have you up and running _Kind switches `kubectl` context to `kind-kind` so you can work with your LOCAL kind cluster using kubectl_

### [Tilt](https://docs.tilt.dev/install.html)

- installing tilt is optional
  - `curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash`
  - once installed run the command `tilt up`, there is a `./Tiltfile` provided, please review and update as you work
