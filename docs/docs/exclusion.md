---
title: Exclusion
---

## Excluding registries, repositories, and images
Eraser can exclude registries (example, `docker.io/library/*`) and also specific images with a tag (example, `docker.io/library/ubuntu:18.04`) or digest (example, `sha256:80f31da1ac7b312ba29d65080fd...`) from its removal process.

To exclude any images or registries from the removal, create a configmap named `excluded` in the eraser-system namespace with a JSON file holding the excluded images.

```bash
$ cat > sample.json <<EOF
{"excluded": ["docker.io/library/*", "ghcr.io/azure/test:latest"]}
EOF

$ kubectl create configmap excluded --from-file=excluded=sample.json --namespace=eraser-system
```
