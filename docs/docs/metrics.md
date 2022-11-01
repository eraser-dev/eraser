---
title: Metrics
---

To view Eraser metrics, you will need to deploy an Open Telemetry collector and an exporter. An example collector is [otelcollector.yaml](../../test/e2e/test-data/otelcollector.yaml), and the endpoint can be passed in when deploying eraser ex: --otel-endpoint=otel-collector:4318. In this example, we are logging the collected data to the otel-collector pod, but a separate exporter can be configured.

Below is the list of metrics provided by Eraser:

#### Eraser
- count
	- name: ImagesRemoved

		- description: Total images removed by eraser

 #### Scanner
- count
	- name: VulnerableImages

		- description: Total vulnerable images detected
  
 #### ImageJob
 - count
	- name: ImageJobTotal
		- description: Total ImageJobs scheduled

	- name: PodsCompleted
		- description: Total pods completed
	-  name: PodsFailed
		- description: Total pods failed
- summary
	- name: ImageJobDuration
		- description: Total time for ImageJobs scheduled to complete
