---
title: "Developer Setup"
date:
draft: false
weight: 305
---

The [Postgres-Operator](https://github.com/crunchydata/postgres-operator) is an open source project hosted on GitHub.

This guide is intended for those wanting to build the Operator from source or contribute via pull requests.


# Prerequisites
The target development host for these instructions is a CentOS 7 or RHEL 7 host. Others operating systems
are possible, however we do not support building or running the Operator on others at this time.

## Environment Variables

The following environment variables are expected by the steps in this guide:

Variable | Example | Description
-------- | ------- | -----------
`GOPATH` | $HOME/odev | Golang project directory
`PGOROOT` | $GOPATH/src/github.com/crunchydata/postgres-operator | Operator repository location
`PGO_BASEOS` | centos7 | Base OS for container images
`PGO_CMD` | kubectl | Cluster management tool executable
`PGO_IMAGE_PREFIX` | crunchydata | Container image prefix
`PGO_OPERATOR_NAMESPACE` | pgo | Kubernetes namespace for the operator
`PGO_VERSION` | 4.3.0 | Operator version

{{% notice tip %}}
`examples/envs.sh` contains the above variable definitions as well as others used by postgres-operator tools
{{% /notice %}}


## Other requirements

* The development host has been created, has access to `yum` updates, and has a regular user account with `sudo` rights to run `yum`.
* `GOPATH` points to a directory containing `src`,`pkg`, and `bin` directories.
* The development host has `$GOPATH/bin` added to its `PATH` environment variable. Development tools will be installed to this path. Defining a `GOBIN` environment variable other than `$GOPATH/bin` may yield unexpected results.
* The development host has `git` installed and has cloned the postgres-operator repository to `$GOPATH/src/github.com/crunchydata/postgres-operator`. Makefile targets below are run from the repository directory.
* Deploying the Operator will require deployment access to a Kubernetes cluster. Clusters built on OpenShift Container Platform (OCP) or built using `kubeadm` are the validation targets for Pull Requests and thus recommended for devleopment. Instructions for setting up these clusters are outside the scope of this guide.


# Building

## Dependencies

Configuring build dependencies is automated via the `setup` target in the project Makefile:

    make setup

The setup target ensures the presence of:

* `GOPATH` and `PATH` as described in the prerequisites
* EPEL yum repository
* golang compiler
* `dep` dependency manager
* NSQ messaging binaries
* `docker` container tool
* `buildah` OCI image building tool
* `expenv` config tool

By default, docker is not configured to run its daemon. Refer to the [docker post-installation instructions](https://docs.docker.com/install/linux/linux-postinstall/) to configure it to run once or at system startup. This is not done automatically.

## Compile

{{% notice tip %}}
Please be sure to have your GPG Key and `.repo` file in the `conf` directory
before proceeding.
{{% /notice %}}

You will build all the Operator binaries and Docker images by running:

    make all

This assumes you have Docker installed and running on your development host.

By default, the Makefile will use buildah to build the container images, to override this default to use docker to build the images, set the IMGBUILDER variable to `docker`


The project uses the golang dep package manager to vendor all the golang source dependencies into the `vendor` directory.  You typically do not need to run any `dep` commands unless you are adding new golang package dependencies into the project outside of what is within the project for a given release.

After a full compile, you will have a `pgo` binary in `$HOME/odev/bin` and the Operator images in your local Docker registry.

## Release
You can perform a release build by running:

    make release

This will compile the Mac and Windows versions of `pgo`.


# Deployment

Now that you have built the Operator images, you can push them to your Kubernetes cluster if that cluster is remote to your development host.

You would then run:

    make deployoperator

To deploy the Operator on your Kubernetes cluster.  If your Kubernetes cluster is not local to your development host, you will need to specify a config file that will connect you to your Kubernetes cluster. See the Kubernetes documentation for details.


# Troubleshooting

Debug level logging in turned on by default when deploying the Operator.

Sample bash functions are supplied in `examples/envs.sh` to view
the Operator logs.

You can view the Operator REST API logs with the `alog` bash function.

You can view the Operator core logic logs with the `olog` bash function.

You can view the Scheduler logs with the `slog` bash function.

These logs contain the following details:

	Timestamp
	Logging Level
	Message Content
	Function Information
	File Information
	PGO version

Additionally, you can view the Operator deployment Event logs with the `elog` bash function.

You can enable the `pgo` CLI debugging with the following flag:

    pgo version --debug

You can set the REST API URL as follows after a deployment if you are
developing on your local host by executing the `setip` bash function.
