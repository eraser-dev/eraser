---
title: Exclusion
---

## Excluding registries, repositories, and images
Eraser can exclude registries (example, `docker.io/library/*`) and also specific images with a tag (example, `docker.io/library/ubuntu:18.04`) or digest (example, `sha256:80f31da1ac7b312ba29d65080fd...`) from its removal process.

To exclude any images or registries from the removal, create configmap(s) with the label `eraser.sh/exclude.list=true` in the eraser-system namespace with a JSON file holding the excluded images.

```bash
$ cat > sample.json <<"EOF"
{
  "excluded": [
    "docker.io/library/*",
    "ghcr.io/eraser-dev/test:latest"
  ]
}
EOF

$ kubectl create configmap excluded --from-file=sample.json --namespace=eraser-system
$ kubectl label configmap excluded eraser.sh/exclude.list=true -n eraser-system
```

## Exempting Nodes from the Eraser Pipeline
Exempting nodes from cleanup was added in v1.0.0. When deploying Eraser, you can specify whether there is a list of nodes you would like to `include` or `exclude` from the cleanup process using the configmap. For more information, see the section on [customization](https://eraser-dev.github.io/eraser/docs/customization).
