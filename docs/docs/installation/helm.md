---
id: helm
title: Helm
---

### Get Repo Info
```
helm repo add eraser https://azure.github.io/eraser/charts
helm repo update
```
_See [helm repo](https://helm.sh/docs/helm/helm_repo/) for command documentation._

### Install Chart
```
helm install -n eraser-system [RELEASE_NAME] eraser/eraser --create-namespace
```

_See [parameters](#parameters) below._

_See [helm install](https://helm.sh/docs/helm/helm_install/) for command documentation._

### Parameters
| Parameter                                     | Description                                                                                                                                                                                                                                         | Default                                                                   |
| :-------------------------------------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------------------------------------------------------------------------ |
| controllerManager.image.repository                              | Image repository                                                                                                                                                                                                                                    | `ghcr.io/azure/eraser-manager`   |
| controllerManager.image.tag                                     | Image tag                                                                                                                                                                                                                                    | Current release version: `v0.1.0` |
| controllerManager.image.pullPolicy                              | Image pull policy                                                                                                      | `Always`   |               
| controllerManager.securityContext                               | Security context applied on the container                                                                                                                                                                                                                | `{ allowPrivilegeEscalation: false }`                                                                      |
| controllerManager.resources                                     | The resource request/limits for the container image                                                                                                                                                                                                 | limits: 0.1 CPU, 30Mi, requests: 0.1 CPU, 20Mi                            |
| controllerManager.nodeSelector                                  | The node selector to use for pod scheduling                                                                                                                                                                                                         | `kubernetes.io/os: linux`                                                 |
| controllerManager.tolerations                                   | The tolerations to use for pod scheduling                                                                                                                                                                                                           | `[]`                                                                      |
| controllerManager.affinity                                      | The node affinity to use for pod scheduling                                                                                                                                                                                                         | `{}`                                                                      |
| eraser.image.repository                        | Image repository for worker                                                                                                                                                                                                                                    | `ghcr.io/azure/eraser`   |
| eraser.image.tag                               | Image tag for worker                                                                                                                                                                                                                                    | Current release version: `v0.1.0` |
| nameOverride                                  | Override name if needed                                                                                                                                                                                                                | `""`                                                                      |