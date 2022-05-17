# Eraser Helm Chart

## Get Repo Info

```console
helm repo add eraser https://azure.github.io/eraser/charts
helm repo update
```

_See [helm repo](https://helm.sh/docs/helm/helm_repo/) for command documentation._

## Install Chart

```console
# Helm install with eraser-system namespace already created
$ helm install -n eraser-system [RELEASE_NAME] eraser/eraser

# Helm install and create namespace
$ helm install -n eraser-system [RELEASE_NAME] eraser/eraser --create-namespace

```

_See [parameters](#parameters) below._

_See [helm install](https://helm.sh/docs/helm/helm_install/) for command documentation._

## Parameters

| Parameter                                     | Description                                                                                                                                                                                                                                         | Default                                                                   |
| :-------------------------------------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------------------------------------------------------------------------ |
| image.repository                              | Image repository                                                                                                                                                                                                                                    | `ghcr.io/azure/eraser-manager`   |
| image.tag                                     | Image tag                                                                                                                                                                                                                                    | Current tag version: `0.1.0`|
| image.pullPolicy                              | Image pull policy                                                                                                      | `Always`   |               
| workerImage.repository                        | Image repository for worker                                                                                                                                                                                                                                    | `ghcr.io/azure/eraser`   |
| workerImage.tag                                     | Image tag for worker                                                                                                                                                                                                                                    | Current tag version: `0.1.0` |
| imagePullSecrets                             | Specify an array of imagePullSecrets                                                                                                                                                                                                                | `[]`                                                                      |    
| nameOverride                             | Override name if needed                                                                                                                                                                                                                | `""`                                                                      |
| serviceAccount.annotations                             | Annotations to add to the service account                                                                                                                                                                                                                | `{}`                                                                      |
| serviceAccount.name                             | The name of the service account to use. If not set, a name is generated using the name template                                                                                                                                                                                                               | `""`                                                                      |
| podAnnotations                             | Annotations to add to the pod                                                                                                                                                                                                                | `{}`                                                                      |
