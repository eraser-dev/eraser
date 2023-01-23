---
title: Architecture
---
At a high level, Eraser has two main modes of operation: manual and automated.

Manual image removal involves supplying a list of images to remove; Eraser then
deploys pods to clean up the images you supplied.

Automated image removal runs on a timer. By default, the automated process
removes images based on the results of a vulnerability scan. The default
vulnerability scanner is Trivy, but others can be provided in its place. Or,
the scanner can be disabled altogether, in which case Eraser acts as a garbage
collector -- it will remove all non-running images in your cluster.

## Manual image cleanup

<img title="manual cleanup" src="/eraser/docs/img/eraser_manual.png" />

## Automated analysis, scanning, and cleanup

<img title="automated cleanup" src="/eraser/docs/img/eraser_timer.png" />
