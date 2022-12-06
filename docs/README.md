# Website

This website is built using [Docusaurus 2](https://docusaurus.io/), a modern static website generator.

### Installation

```
$ yarn
```

### Local Development

```
$ yarn start
```

This command starts a local development server and opens up a browser window. Most changes are reflected live without having to restart the server.

### Build

```
$ yarn build
```

This command generates static content into the `build` directory and can be served using any static contents hosting service.

### Deployment

Using SSH:

```
$ USE_SSH=true yarn deploy
```

Not using SSH:

```
$ GIT_USER=<Your GitHub username> yarn deploy
```

If you are using GitHub pages for hosting, this command is a convenient way to build the website and push to the `gh-pages` branch.

## Search

Eraser docs website uses Algolia DocSearch service. Please see [here](https://docusaurus.io/docs/search) for more information.

If the search index has any issues:

1. Go to [Algolia search dashboard](https://www.algolia.com/apps/X8MU4GEC0G/explorer/browse/eraser)
1. Click manage index and export configuration
1. Delete the index
1. Import saved configuration
1. Go to [Algolia crawler](https://crawler.algolia.com/admin/crawlers/acc2bdb5-4780-433f-a3e9-bb3b49598320/overview) and restart crawling manually (takes about a few minutes to crawl). This is scheduled to run every week automatically.
