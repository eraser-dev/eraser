---
title: Custom Scanner
---

## Creating a Custom Scanner
To create a custom scanner for non-compliant images, provide your scanner image to Eraser in deployment.

In order for the custom scanner to communicate with the collector and eraser containers, utilize `ReadCollectScanPipe()` to get the list of all non-running images to scan from collector. Then, use `WriteScanErasePipe()` to pass the images found non-compliant by your scanner to eraser for removal. Both functions can be found in [util](../../../pkg/utils/utils.go).
