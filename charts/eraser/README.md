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
|                                              |                                                                                                                                                                                                                |                                                                     |

