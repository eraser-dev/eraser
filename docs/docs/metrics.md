---
title: Metrics
---

To view Eraser metrics, you will need to deploy an Open Telemetry collector in the 'eraser-system' namespace, and an exporter. An example collector with a Prometheus exporter is [otelcollector.yaml](../../test/e2e/test-data/otelcollector.yaml), and the endpoint can be passed in when deploying eraser ex: --otlp-endpoint=otel-collector:4318. In this example, we are logging the collected data to the otel-collector pod, and exporting metrics through Prometheus at 'http://localhost:8889/metrics', but a separate exporter can also be configured.

Below is the list of metrics provided by Eraser:

#### Eraser
- count
	- name: images_removed_total

		- description: Total images removed by eraser

 #### Scanner
- count
	- name: vulnerable_images_total

		- description: Total vulnerable images detected
  
 #### ImageJob
 - count
	- name: imagejob_total
		- description: Total ImageJobs scheduled

	- name: pods_completed_total
		- description: Total pods completed
	-  name: pods_failed_total
		- description: Total pods failed
- summary
	- name: imagejob_duration_seconds
		- description: Total time for ImageJobs scheduled to complete
