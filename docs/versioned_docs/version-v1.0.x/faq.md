---
title: FAQ
---
## Why am I still seeing vulnerable images?
Eraser currently targets **non-running** images, so any vulnerable images that are currently running will not be removed. In addition, the default vulnerability scanning with Trivy removes images with `CRITICAL` vulnerabilities. Any images with lower vulnerabilities will not be removed. This can be configured using the [configmap](https://eraser-dev.github.io/eraser/docs/customization#scanner-options).
