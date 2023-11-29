# GitHub Workflows

This directory contains all of our workflows used in our GitHub CI/CD pipeline.

## Descriptions

### [Scan Images for Vulnerabilities (Trivy)](scan-images.yaml)
Our images are scheduled to be scanned for vulnerabilities using Trivy every Monday at 07:00 UTC.

#### Weekly Scans
By default, our images are built from the `main` branch, and any vulnerabilities caught are published in the [Github Security tab](https://github.com/eraser-dev/eraser/security).

#### Dispatching a Scan
We can do a manual dispatch of the workflow and specify the released version to scan, e.g. `v1.3.0-beta.0`. If left blank, the image will be built off of the branch the workflow is dispatched from.

If we want to publish those results to our [Github Security tab](https://github.com/eraser-dev/eraser/security), we need to toggle the `upload-results` input to `true`.

#### Scan Results
The scan results are automatically stored in the run artifacts. Those can be accessed by going into the workflow run, and under the run's **Summary** there is an **Artifacts** section storing all the images' scan results.

If the `upload-results` input is set to `true`, any vulnerabilities found will be published in the [Github Security tab](https://github.com/eraser-dev/eraser/security).
