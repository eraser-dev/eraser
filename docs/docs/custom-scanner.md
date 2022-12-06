---
title: Custom Scanner
---

## Creating a Custom Scanner
To create a custom scanner for non-compliant images, first retrieve the list of all non-running and non-excluded images from the collector container through the `ReadCollectScanPipe()` function. Process these images with your customized scanner and threshold, and use `WriteScanErasePipe()` to pass the images found non-compliant to the eraser container for removal. Both functions can be found in [util](../../pkg/utils/utils.go), and a boilerplate code can be found [here](../../pkg/scanners/template/scanner_template.go). When complete, provide your custom scanner image to Eraser in deployment.
