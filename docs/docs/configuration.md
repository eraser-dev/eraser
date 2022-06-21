---
sidebar_position: 4
---

# Configuration

There are a series of startup flags that can be used to configure Eraser. Usage documentation for all of them can be found below.

|Name   |Description   |Default   |
|---|---|---|
|`--metrics-bind-address`   |The address the metric endpoint binds to   |`:8080`   |
|`--health-probe-bind-address`   |The address the probe endpoint binds to   |`:8081`   |
|`--leader-elect` | Enable leader election for controller manager | `false` |