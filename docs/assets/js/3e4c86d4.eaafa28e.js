"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[8180],{9424:e=>{e.exports=JSON.parse('{"pluginId":"default","version":"v1.1.x","label":"v1.1.x","banner":"unmaintained","badge":true,"noIndex":false,"className":"docs-version-v1.1.x","isLast":false,"docsSidebars":{"sidebar":[{"type":"link","label":"Introduction","href":"/eraser/docs/v1.1.x/","docId":"introduction"},{"type":"link","label":"Installation","href":"/eraser/docs/v1.1.x/installation","docId":"installation"},{"type":"link","label":"Quick Start","href":"/eraser/docs/v1.1.x/quick-start","docId":"quick-start"},{"type":"link","label":"Architecture","href":"/eraser/docs/v1.1.x/architecture","docId":"architecture"},{"type":"category","label":"Topics","collapsible":true,"collapsed":false,"items":[{"type":"link","label":"Manual Removal","href":"/eraser/docs/v1.1.x/manual-removal","docId":"manual-removal"},{"type":"link","label":"Exclusion","href":"/eraser/docs/v1.1.x/exclusion","docId":"exclusion"},{"type":"link","label":"Customization","href":"/eraser/docs/v1.1.x/customization","docId":"customization"},{"type":"link","label":"Metrics","href":"/eraser/docs/v1.1.x/metrics","docId":"metrics"}]},{"type":"category","label":"Development","collapsible":true,"collapsed":false,"items":[{"type":"link","label":"Setup","href":"/eraser/docs/v1.1.x/setup","docId":"setup"},{"type":"link","label":"Releasing","href":"/eraser/docs/v1.1.x/releasing","docId":"releasing"}]},{"type":"category","label":"Scanning","collapsible":true,"collapsed":false,"items":[{"type":"link","label":"Custom Scanner","href":"/eraser/docs/v1.1.x/custom-scanner","docId":"custom-scanner"},{"type":"link","label":"Trivy","href":"/eraser/docs/v1.1.x/trivy","docId":"trivy"}]},{"type":"link","label":"FAQ","href":"/eraser/docs/v1.1.x/faq","docId":"faq"},{"type":"link","label":"Contributing","href":"/eraser/docs/v1.1.x/contributing","docId":"contributing"},{"type":"link","label":"Code of Conduct","href":"/eraser/docs/v1.1.x/code-of-conduct","docId":"code-of-conduct"}]},"docs":{"architecture":{"id":"architecture","title":"Architecture","description":"At a high level, Eraser has two main modes of operation: manual and automated.","sidebar":"sidebar"},"code-of-conduct":{"id":"code-of-conduct","title":"Code of Conduct","description":"This project has adopted the CNCF Code of Conduct.","sidebar":"sidebar"},"contributing":{"id":"contributing","title":"Contributing","description":"There are several ways to get involved with Eraser","sidebar":"sidebar"},"custom-scanner":{"id":"custom-scanner","title":"Custom Scanner","description":"Creating a Custom Scanner","sidebar":"sidebar"},"customization":{"id":"customization","title":"Customization","description":"Overview","sidebar":"sidebar"},"exclusion":{"id":"exclusion","title":"Exclusion","description":"Excluding registries, repositories, and images","sidebar":"sidebar"},"faq":{"id":"faq","title":"FAQ","description":"Why am I still seeing vulnerable images?","sidebar":"sidebar"},"installation":{"id":"installation","title":"Installation","description":"Manifest","sidebar":"sidebar"},"introduction":{"id":"introduction","title":"Introduction","description":"When deploying to Kubernetes, it\'s common for pipelines to build and push images to a cluster, but it\'s much less common for these images to be cleaned up. This can lead to accumulating bloat on the disk, and a host of non-compliant images lingering on the nodes.","sidebar":"sidebar"},"manual-removal":{"id":"manual-removal","title":"Manual Removal","description":"Create an ImageList and specify the images you would like to remove. In this case, the image docker.io/library/alpine:3.7.3 will be removed.","sidebar":"sidebar"},"metrics":{"id":"metrics","title":"Metrics","description":"To view Eraser metrics, you will need to deploy an Open Telemetry collector in the \'eraser-system\' namespace, and an exporter. An example collector with a Prometheus exporter is otelcollector.yaml, and the endpoint can be specified using the configmap. In this example, we are logging the collected data to the otel-collector pod, and exporting metrics through Prometheus at \'http8889/metrics\', but a separate exporter can also be configured.","sidebar":"sidebar"},"quick-start":{"id":"quick-start","title":"Quick Start","description":"This tutorial demonstrates the functionality of Eraser and validates that non-running images are removed succesfully.","sidebar":"sidebar"},"releasing":{"id":"releasing","title":"Releasing","description":"Create Release Pull Request","sidebar":"sidebar"},"setup":{"id":"setup","title":"Setup","description":"This document describes the steps to get started with development.","sidebar":"sidebar"},"trivy":{"id":"trivy","title":"Trivy","description":"Trivy Provider Options","sidebar":"sidebar"}}}')}}]);