---
title: Trivy
---

## Trivy Provider Options
The Trivy provider is used in Eraser for image scanning and detecting vulnerabilities. The scanner will first look for the trivy executable at `/trivy` (the default path in the container), and if not found, will fall back to searching for `trivy` in the system PATH. See [Customization](https://eraser-dev.github.io/eraser/docs/customization#scanner-options) for more details on configuring the scanner.
