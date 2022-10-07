---
title: Customization
---

By default, successful jobs will be deleted after a period of time. You can change this behavior by setting the following flags in the eraser-controller-manager:

- `--job-cleanup-on-success-delay`: Duration to delay job deletion after successful runs. 0 means no delay. Defaults to `0`.
- `--job-cleanup-on-error-delay`: Duration to delay job deletion after errored runs. 0 means no delay. Defaults to `24h`. 
- `--job-success-ratio`: Ratio of successful/total runs to consider a job successful. 1.0 means all runs must succeed. Defaults to `1.0`.  

For duration, valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
