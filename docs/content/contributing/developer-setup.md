---
title: "Development Environment"
date:
draft: false
weight: 305
---

The [PostgreSQL Operator](https://github.com/crunchydata/postgres-operator) is an open source project hosted on GitHub.

This guide is intended for those wanting to build the Operator from source or contribute via pull requests.


# Prerequisites

The target development host for these instructions is a CentOS 8 or RHEL 8 host. Others operating systems
are possible, however we do not support building or running the Operator on others at this time.

## Environment Variables

The following environment variables are expected by the steps in this guide:

Variable | Example | Description
-------- | ------- | -----------
`PGOROOT` | $HOME/postgres-operator | Operator repository location
`PGO_CONF_DIR` | $PGOROOT/installers/ansible/roles/pgo-operator/files | Operator Config Template Directory

{{% notice tip %}}
`examples/envs.sh` contains the above variable definitions as well as others used by postgres-operator tools
{{% /notice %}}


## Other requirements

* The development host has `git` installed and has cloned the [postgres-operator](https://github.com/CrunchyData/postgres-operator.git) repository. Makefile targets below are run from the repository directory.
* Deploying the Operator will require deployment access to a Kubernetes or OpenShift cluster
* Once you have cloned the git repository, you will need to download the CentOS repository files and GPG keys and place them in the `$PGOROOT/conf` directory. You can do so with the following code:

```shell
cd $PGOROOT
curl https://api.developers.crunchydata.com/downloads/repo/rpm-centos/postgresql12/crunchypg12.repo > conf/crunchypg12.repo
curl https://api.developers.crunchydata.com/downloads/repo/rpm-centos/postgresql11/crunchypg11.repo > conf/crunchypg11.repo
curl https://api.developers.crunchydata.com/downloads/gpg/RPM-GPG-KEY-crunchydata-dev > conf/RPM-GPG-KEY-crunchydata-dev
```

# Building

## Dependencies

Configuring build dependencies is automated via the `setup` target in the project Makefile:

    make setup

The setup target ensures the presence of:

* [`go`](https://golang.org/) compiler version 1.13+
* `buildah` OCI image building tool version 1.14.9+

## Code Generation

Code generation is leveraged to generate the clients and informers utilized to interact with the
various [Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
(e.g. `pgclusters`) comprising the PostgreSQL Operator declarative API.  Code generation is provided
by the [Kubernetes code-generator project](https://github.com/kubernetes/code-generator),
and the following Make target is included within the PostgreSQL Operator project to update that code
as needed:

```bash
# Update any generated code:
make generate
```

Therefore, in the event that a Custom Resource defined within the PostgreSQL Operator API
(`$PGOROOT/pkg/apis/crunchydata.com`) is updated, the `verify-codegen` target will indicate that
an update is needed, and the `update-codegen` target should then be utilized to generate the
updated code prior to compiling.

## Compile

{{% notice tip %}}
Please be sure to have your GPG Key and `.repo` file in the `conf` directory
before proceeding.
{{% /notice %}}

You will build all the Operator binaries and Docker images by running:

    make all

This assumes you have Docker installed and running on your development host.

By default, the Makefile will use buildah to build the container images, to override this default to use docker to build the images, set the IMGBUILDER variable to `docker`


After a full compile, you will have a `pgo` binary in `$PGOROOT/bin` and the Operator images in your local Docker registry.

# Deployment

Now that you have built the PostgreSQL Operator images, you can now deploy them
to your Kubernetes cluster by following the [Bash Installation Guide]({{< relref "installation/other/bash.md" >}}).

# Testing

Once the PostgreSQL Operator is deployed, you can run the end-to-end regression
test suite interface with the PostgreSQL client. You need to ensure
that the `pgo` client executable is in your `$PATH`. The test suite can be run
using the following commands:

```shell
cd $PGOROOT/testing/pgo_cli
GO111MODULE=on go test -count=1 -parallel=2 -timeout=30m -v .
```

For more information, please follow the [testing README](https://github.com/CrunchyData/postgres-operator/blob/master/testing/pgo_cli/README.md)
in the source repository.

# Troubleshooting

Debug level logging in turned on by default when deploying the Operator.

Sample bash functions are supplied in `examples/envs.sh` to view
the Operator logs.

You can view the Operator REST API logs with the `alog` bash function.

You can view the Operator core logic logs with the `olog` bash function.

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
