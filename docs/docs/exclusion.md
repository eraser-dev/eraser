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

## Exempting Nodes from the Eraser Pipeline

When deploying Eraser, you can specify whether there is a list of nodes you would like to `include` or `exclude` from the cleanup process using the `--filter-nodes` argument.  

Nodes with the selector `eraser.sh/cleanup.filter` will be filtered accordingly. 
- If `include` is provided, eraser and collector pods will only be scheduled on nodes with the selector `eraser.sh/cleanup.filter`. 
- If `exclude` is provided, eraser and collector pods will be scheduled on all nodes besides those with the selector `eraser.sh/cleanup.filter`.

Unless specified, the default value of `--filter-nodes` is `exclude`. Because Windows nodes are not supported, they will always be excluded regardless of the `eraser.sh/cleanup.filter` label or the value of `--filter-nodes`.

Additional node selectors can be provided through the `--filter-nodes-selector` flag.
