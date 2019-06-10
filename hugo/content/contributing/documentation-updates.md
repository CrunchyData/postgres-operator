---
title: "Updating Documentation"
date:
draft: false
weight: 901
---

## Documentation

The [documentation website](/) is generated using [Hugo](https://gohugo.io/).

## Hosting Hugo Locally (Optional)

If you would like to build the documentation locally, view the
[official Installing Hugo](https://gohugo.io/getting-started/installing/) guide to set up Hugo locally.

You can then start the server by running the following commands -

```
cd $COROOT/hugo/
hugo server
```

The local version of the Hugo server is accessible by default from
*localhost:1313*. Once you've run *hugo server*, that will let you interactively make changes to the documentation as desired and view the updates
in real-time.

## Contributing to the Documentation

All documentation is in Markdown format and uses Hugo weights for positioning of the pages.

The current production release documentation is updated for every tagged major release.

When you're ready to commit a change, please verify that the documentation generates locally.
