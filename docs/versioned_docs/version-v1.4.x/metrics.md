---
title: Metrics
---

To view Eraser metrics, you will need to deploy an Open Telemetry collector in the 'eraser-system' namespace, and an exporter. An example collector with a Prometheus exporter is [otelcollector.yaml](https://github.com/eraser-dev/eraser/blob/main/test/e2e/test-data/otelcollector.yaml), and the endpoint can be specified using the [configmap](https://eraser-dev.github.io/eraser/docs/customization#universal-options). In this example, we are logging the collected data to the otel-collector pod, and exporting metrics through Prometheus at 'http://localhost:8889/metrics', but a separate exporter can also be configured.

Below is the list of metrics provided by Eraser per run:

#### Eraser
```yaml
- count
	- name: images_removed_run_total
		- description: Total images removed by eraser
```

 #### Scanner
 ```yaml
- count
	- name: vulnerable_images_run_total
		- description: Total vulnerable images detected
 ```

 #### ImageJob
 ```yaml
 - count
	- name: imagejob_run_total
		- description: Total ImageJobs scheduled
	- name: pods_completed_run_total
		- description: Total pods completed
	-  name: pods_failed_run_total
		- description: Total pods failed
- summary
	- name: imagejob_duration_run_seconds
		- description: Total time for ImageJobs scheduled to complete
```
