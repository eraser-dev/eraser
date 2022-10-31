---
title: Metrics
---

To view Eraser metrics, you will need to deploy an Open Telemetry collector and an exporter. An example collector is [otelcollector.yaml](../../test/e2e/test-data/otelcollector.yaml), and the endpoint can be passed in when deploying eraser ex: --otel-endpoint=otel-collector:4318. In this example, we are logging the collected data to the collector pod, but a separate exporter can be configured.

Metrics recorded are total images removed and total vulnerable images found. For ImageJobs, total duration, pods completed, pods failed, and total jobs scheduled are also recorded.