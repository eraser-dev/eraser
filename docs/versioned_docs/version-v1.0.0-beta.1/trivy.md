---
title: Trivy
---

## Trivy Provider Options
The trivy provider is used in Eraser for image scanning and detecting vulnerabilities. The following arguments can be supplied to the scanner to specify which types of images will be detected for removal by the trivy scanner container:

* --ignore-unfixed: boolean to report only fixed vulnerabilities (default true)
* --security-checks: comma-separated list of what security issues to detect (default "vuln")
* --vuln-type: list of severity levels to report  (default "CRITICAL")
* --delete-scan-failed-images : boolean to delete images for which scanning has failed (default true)
