---
title: Releasing
---

## Create Release Pull Request

1. Go to `create_release_pull_request` workflow under actions.
2. Select run workflow, and use the workflow from your branch. 
3. Input release version with the semantic version identifying the release.
4. Click run workflow and review the PR created by github-actions.

# Releasing

5. Once the PR is merged to `main`, tag that commit with release version and push tags to remote repository.

   ```
   git checkout <BRANCH NAME>
   git pull origin <BRANCH NAME>
   git tag -a <NEW VERSION> -m '<NEW VERSION>'
   git push origin <NEW VERSION>
   ```
6. Pushing the release tag will trigger GitHub Actions to trigger `release` job.
   This will build the `ghcr.io/eraser-dev/remover`, `ghcr.io/eraser-dev/eraser-manager`, `ghcr.io/eraser-dev/collector`, and `ghcr.io/eraser-dev/eraser-trivy-scanner` images automatically, then publish the new release tag.

## Publishing

1. GitHub Action will create a new release, review and edit it at https://github.com/eraser-dev/eraser/releases

## Notifying

1. Send an email to the [Eraser mailing list](https://groups.google.com/g/eraser-dev) announcing the release, with links to GitHub.
2. Post a message on the [Eraser Slack channel](https://kubernetes.slack.com/archives/C03Q8KV8YQ4) with the same information.