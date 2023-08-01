---
title: Customization
---

## Overview

Eraser uses a configmap to configure its behavior. The configmap is part of the
deployment and it is not necessary to deploy it manually. Once deployed, the configmap
can be edited at any time:

```bash
kubectl edit configmap --namespace eraser-system eraser-manager-config
```

If an eraser job is already running, the changes will not take effect until the job completes.
The configuration is in yaml.

## Key Concepts

### Basic architecture

The _manager_ runs as a pod in your cluster and manages _ImageJobs_. Think of
an _ImageJob_ as a unit of work, performed on every node in your cluster. Each
node runs a sub-job. The goal of the _ImageJob_ is to assess the images on your
cluster's nodes, and to remove the images you don't want. There are two stages:
1. Assessment
1. Removal.


### Scheduling

An _ImageJob_ can either be created on-demand (see [Manual Removal](https://eraser-dev.github.io/eraser/docs/manual-removal)),
or they can be spawned on a timer like a cron job. On-demand jobs skip the
assessment stage and get right down to the business of removing the images you
specified. The behavior of an on-demand job is quite different from that of
timed jobs.

### Fault Tolerance

Because an _ImageJob_ runs on every node in your cluster, and the conditions on
each node may vary widely, some of the sub-jobs may fail. If you cannot
tolerate any failure, set the `manager.imageJob.successRatio` property to
`1.0`. If 75% success sounds good to you, set it to `0.75`. In that case, if
fewer than 75% of the pods spawned by the _ImageJob_ report success, the job as
a whole will be marked as a failure.

This is mainly to help diagnose error conditions. As such, you can set
`manager.imageJob.cleanup.delayOnFailure` to a long value so that logs can be
captured before the spawned pods are cleaned up.

### Excluding Nodes

For various reasons, you may want to prevent Eraser from scheduling pods on
certain nodes. To do so, the nodes can be given a special label. By default,
this label is `eraser.sh/cleanup.filter`, but you can configure the behavior with
the options under `manager.nodeFilter`. The [table](#detailed-options) provides more detail.

### Configuring Components

An _ImageJob_ is made up of various sub-jobs, with one sub-job for each node.
These sub-jobs can be broken down further into three stages.
1. Collection (What is on the node?)
1. Scanning (What images conform to the policy I've provided?)
1. Removal (Remove images based on the results of the above)

Of the above stages, only Removal is mandatory. The others can be disabled.
Furthermore, manually triggered _ImageJobs_ will skip right to removal, even if
Eraser is configured to collect and scan. Collection and Scanning will only
take place when:
1. The collector and/or scanner `components` are enabled, AND
1. The job was *not* triggered manually by creating an _ImageList_.

### Swapping out components

The collector, scanner, and eraser components can all be swapped out. This
enables you to build and host the images yourself. In addition, the scanner's
behavior can be completely tailored to your needs by swapping out the default
image with one of your own. To specify the images, use the
`components.<component>.image.repo` and `components.<component>.image.tag`,
where `<component>` is one of `collector`, `scanner`, or `eraser`.

## Universal Options

The following portions of the configmap apply no matter how you spawn your
_ImageJob_. The values provided below are the defaults. For more detail on
these options, see the [table](#detailed-options).

```yaml
manager:
  runtime: containerd
  otlpEndpoint: "" # empty string disables OpenTelemetry
  logLevel: info
  profile:
    enabled: false
    port: 6060
  imageJob:
    successRatio: 1.0
    cleanup:
      delayOnSuccess: 0s
      delayOnFailure: 24h
  pullSecrets: [] # image pull secrets for collector/scanner/eraser
  priorityClassName: "" # priority class name for collector/scanner/eraser
  nodeFilter:
    type: exclude # must be either exclude|include
    selectors:
      - eraser.sh/cleanup.filter
      - kubernetes.io/os=windows
components:
  eraser:
    image:
      repo: ghcr.io/eraser-dev/eraser
      tag: v1.0.0
    request:
      mem: 25Mi
      cpu: 0
    limit:
      mem: 30Mi
      cpu: 1000m
```

## Component Options

```yaml
components:
  collector:
    enabled: true
    image:
      repo: ghcr.io/eraser-dev/collector
      tag: v1.0.0
    request:
      mem: 25Mi
      cpu: 7m
    limit:
      mem: 500Mi
      cpu: 0
  scanner:
    enabled: true
    image:
      repo: ghcr.io/eraser-dev/eraser-trivy-scanner
      tag: v1.0.0
    request:
      mem: 500Mi
      cpu: 1000m
    limit:
      mem: 2Gi
      cpu: 0
    config: |
      # this is the schema for the provided 'trivy-scanner'. custom scanners
      # will define their own configuration. see the below
  eraser:
    image:
      repo: ghcr.io/eraser-dev/eraser
      tag: v1.0.0
    request:
      mem: 25Mi
      cpu: 0
    limit:
      mem: 30Mi
      cpu: 1000m
```

## Scanner Options

These options can be provided to `components.scanner.config`. They will be
passed through  as a string to the scanner container and parsed there. If you
want to configure your own scanner, you must provide some way to parse this.

Below are the values recognized by the provided `eraser-trivy-scanner` image.
Values provided below are the defaults.

```yaml
cacheDir: /var/lib/trivy # The file path inside the container to store the cache
dbRepo: ghcr.io/aquasecurity/trivy-db # The container registry from which to fetch the trivy database
deleteFailedImages: true # if true, remove images for which scanning fails, regardless of why it failed
vulnerabilities:
  ignoreUnfixed: true # consider the image compliant if there are no known fixes for the vulnerabilities found.
  types: # a list of vulnerability types. for more info, see trivy's documentation.
    - os
    - library
  securityChecks: # see trivy's documentation for more invormation
    - vuln
  severities: # in this case, only flag images with CRITICAL vulnerability for removal
    - CRITICAL
timeout:
  total: 23h # if scanning isn't completed before this much time elapses, abort the whole scan
  perImage: 1h # if scanning a single image exceeds this time, scanning will be aborted
```

## Detailed Options

| Option | Description | Default |
| --- | --- | --- |
| manager.runtime | The runtime to use for the manager's containers. Must be one of containerd, crio, or dockershim. It is assumed that your nodes are all using the same runtime, and there is currently no way to configure multiple runtimes. | containerd |
| manager.otlpEndpoint | The endpoint to send OpenTelemetry data to. If empty, data will not be sent. | "" |
| manager.logLevel | The log level for the manager's containers. Must be one of debug, info, warn, error, dpanic, panic, or fatal. | info |
| manager.scheduling.repeatInterval | Use only when collector ando/or scanner are enabled. This is like a cron job, and will spawn an _ImageJob_ at the interval provided. | 24h |
| manager.scheduling.beginImmediately | If set to true, the fist _ImageJob_ will run immediately. If false, the job will not be spawned until after the interval (above) has elapsed. | true |
| manager.profile.enabled | Whether to enable profiling for the manager's containers. This is for debugging with `go tool pprof`. | false |
| manager.profile.port | The port on which to expose the profiling endpoint. | 6060 |
| manager.imageJob.successRatio | The ratio of successful image jobs required before a cleanup is performed. | 1.0 |
| manager.imageJob.cleanup.delayOnSuccess | The amount of time to wait after a successful image job before performing cleanup. | 0s |
| manager.imageJob.cleanup.delayOnFailure | The amount of time to wait after a failed image job before performing cleanup. | 24h |
| manager.pullSecrets | The image pull secrets to use for collector, scanner, and eraser containers. | [] |
| manager.priorityClassName | The priority class to use for collector, scanner, and eraser containers. | "" |
| manager.nodeFilter.type | The type of node filter to use. Must be either "exclude" or "include". | exclude |
| manager.nodeFilter.selectors | A list of selectors used to filter nodes. | [] |
| components.collector.enabled | Whether to enable the collector component. | true |
| components.collector.image.repo | The repository containing the collector image. | ghcr.io/eraser-dev/collector |
| components.collector.image.tag | The tag of the collector image. | v1.0.0 |
| components.collector.request.mem | The amount of memory to request for the collector container. | 25Mi |
| components.collector.request.cpu | The amount of CPU to request for the collector container. | 7m |
| components.collector.limit.mem | The maximum amount of memory the collector container is allowed to use. | 500Mi |
| components.collector.limit.cpu | The maximum amount of CPU the collector container is allowed to use. | 0 |
| components.scanner.enabled | Whether to enable the scanner component. | true |
| components.scanner.image.repo | The repository containing the scanner image. | ghcr.io/eraser-dev/eraser-trivy-scanner |
| components.scanner.image.tag | The tag of the scanner image. | v1.0.0 |
| components.scanner.request.mem | The amount of memory to request for the scanner container. | 500Mi |
| components.scanner.request.cpu | The amount of CPU to request for the scanner container. | 1000m |
| components.scanner.limit.mem | The maximum amount of memory the scanner container is allowed to use. | 2Gi |
| components.scanner.limit.cpu | The maximum amount of CPU the scanner container is allowed to use. | 0 |
| components.scanner.config | The configuration to pass to the scanner container, as a YAML string. | See YAML below |
| components.eraser.image.repo | The repository containing the eraser image. | ghcr.io/eraser-dev/eraser |
| components.eraser.image.tag | The tag of the eraser image. | v1.0.0 |
| components.eraser.request.mem | The amount of memory to request for the eraser container. | 25Mi |
| components.eraser.request.cpu | The amount of CPU to request for the eraser container. | 0 |
