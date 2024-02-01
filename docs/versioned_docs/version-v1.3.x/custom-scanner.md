---
title: Custom Scanner
---

## Creating a Custom Scanner
To create a custom scanner for non-compliant images, use the following [template](https://github.com/eraser-dev/eraser-scanner-template/).

In order to customize your scanner, start by creating a `NewImageProvider()`. The ImageProvider interface can be found can be found [here](../../pkg/scanners/template/scanner_template.go). 

The ImageProvider will allow you to retrieve the list of all non-running and non-excluded images from the collector container through the `ReceiveImages()` function. Process these images with your customized scanner and threshold, and use `SendImages()` to pass the images found non-compliant to the eraser container for removal. Finally, complete the scanning process by calling `Finish()`.

When complete, provide your custom scanner image to Eraser in deployment.
