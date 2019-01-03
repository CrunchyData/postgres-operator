
---
title: "Contributing"
date: {docdate}
draft: false
---

## Getting Started

Welcome! Thank you for your interest in contributing. Before submitting a new [issue](https://github.com/CrunchyData/postgres-operator/issues/new)
or [pull request](https://github.com/CrunchyData/postgres-operator/pulls) to the [Crunchy Data
PostgreSQL Operator](https://github.com/CrunchyData/postgres-operator/) project on GitHub, *please review any open or closed issues* [here](https://github.com/crunchydata/postgres-operator/issues)
in addition to any existing open pull requests.

## Documentation

The documentation website (located at https://crunchydata.github.io/postgres-operator/) is generated using [Hugo](https://gohugo.io/) and
[GitHub Pages](https://pages.github.com/). If you would like to build the documentation locally, view the
[official Installing Hugo](https://gohugo.io/getting-started/installing/) guide to set up Hugo locally. You can then start the server by
running the following commands -


cd $COROOT/hugo/<br>
vi config.toml<br>
hugo server<br>


When you edit *config.toml*, you'll set `baseURL = "/"`. This will make the local version of the Hugo server accessible by default from
`localhost:1313`. Once you've run `hugo server`, that will let you interactively make changes to the documentation as desired and view the updates
in real-time.

*When you're ready to commit a change*, you only need to check in the files in the hugo folder.