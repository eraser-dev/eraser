---
title: FAQ
---
## Why am I still seeing vulnerable images?
Eraser currently targets **non-running** images, so any vulnerable images that are currently running will not be removed. In addition, the default vulnerability scanning with Trivy removes images with `CRITICAL` vulnerabilities. Any images with lower vulnerabilities will not be removed. This can be configured using the [configmap](https://eraser-dev.github.io/eraser/docs/customization#scanner-options).

## How is Eraser different from Kubernetes garbage collection?
The native garbage collection in Kubernetes works a bit differently than Eraser. By default, garbage collection begins when disk usage reaches 85%, and stops when it gets down to 80%. More details about Kubernetes garbage collection can be found in the [Kubernetes documentation](https://kubernetes.io/docs/concepts/architecture/garbage-collection/), and configuration options can be found in the [Kubelet documentation](https://kubernetes.io/docs/reference/config-api/kubelet-config.v1beta1/). 

There are a couple core benefits to using Eraser for image cleanup:
* Eraser can be configured to use image vulnerability data when making determinations on image removal
* By interfacing directly with the container runtime, Eraser can clean up images that are not managed by Kubelet and Kubernetes
