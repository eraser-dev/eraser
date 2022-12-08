---
title: Introduction
slug: /
---

# Introduction

When deploying to Kubernetes, it's common for pipelines to build and push images to a cluster, but it's much less common for these images to be cleaned up. This can lead to accumulating bloat on the disk, and a host of non-compliant images lingering on the nodes.

The current garbage collection process deletes images based on a percentage of load, but this process does not consider the vulnerability state of the images. **Eraser** aims to provide a simple way to determine the state of an image, and delete it if it meets the specified criteria.
