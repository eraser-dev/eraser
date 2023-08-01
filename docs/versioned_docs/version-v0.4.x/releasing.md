---
title: Releasing
---

## Overview

The release process consists of three phases: versioning, building, and publishing.

Versioning involves maintaining the following files:

- **Makefile** - the Makefile contains a VERSION variable that defines the version of the project.
- **manager.yaml** - the controller-manager deployment yaml contains the latest release tag image of the project.
- **eraser.yaml** - the eraser.yaml contains all eraser resources to be deployed to a cluster including the latest release tag image of the project.

The steps below explain how to update these files. In addition, the repository should be tagged with the semantic version identifying the release.

Building involves obtaining a copy of the repository and triggering a build as part of the GitHub Actions CI pipeline.

Publishing involves creating a release tag and creating a new _Release_ on GitHub.

## Versioning

1. Obtain a copy of the repository.

   ```
   git clone git@github.com:eraser-dev/eraser.git
   ```

1. If this is a patch release for a release branch, check out applicable branch, such as `release-0.1`. If not, branch should be `main`

1. Execute the release-patch target to generate patch. Give the semantic version of the release:

   ```
   make release-manifest NEWVERSION=vX.Y.Z
   ```

1. Promote staging manifest to release.

   ```
   make promote-staging-manifest
   ```

1. If it's a new minor release (e.g. v0.**4**.x -> 0.**5**.0), tag docs to be versioned. Make sure to keep patch version as `.x` for a minor release.

	```
	make version-docs NEWVERSION=v0.5.x
	```

1. Preview the changes:

   ```
   git status
   git diff
   ```

## Building and releasing

1. Commit the changes and push to remote repository to create a pull request.

   ```
   git checkout -b release-<NEW VERSION>
   git commit -a -s -m "Prepare <NEW VERSION> release"
   git push <YOUR FORK>
   ```

2. Once the PR is merged to `main` or `release` branch (`<BRANCH NAME>` below), tag that commit with release version and push tags to remote repository.

   ```
   git checkout <BRANCH NAME>
   git pull origin <BRANCH NAME>
   git tag -a <NEW VERSION> -m '<NEW VERSION>'
   git push origin <NEW VERSION>
   ```

3. Pushing the release tag will trigger GitHub Actions to trigger `release` job.
   This will build the `ghcr.io/eraser-dev/eraser` and `ghcr.io/eraser-dev/eraser-manager` images automatically, then publish the new release tag.

## Publishing

1. GitHub Action will create a new release, review and edit it at https://github.com/eraser-dev/eraser/releases
