
---
title: "Develop"
date: {docdate}
draft: false

weight: 60
---

# Developing

The [Postgres-Operator](https://github.com/crunchydata/postgres-operator) is an open source project hosted on github.

Developers that wish to build the Operator from source or contribute to the project via pull requests would set up a development environment as follows:

## Create Kubernetes Cluster 
We use either Openshift Container Platform or kubeadm to install our development clusters,  we currently develop and test on CentOS and RHEL hosts.  Each of those installation methods are documented on the respective project's sites.

## Create a Local Development Host
We currently build on CentoOS and RHEL, others are possible however we don't support other Linux variants currently.

## Perform Manual Install
You can follow the manual installation method described in this documentation to make sure you can deploy from your local development host to your Kubernetes cluster.

## Build Locally
You can now build the Operator from source on local on your development host.  Here are some steps to follow:

### Get Build Dependencies

    make setup

That target will install a golang compiler, and any other build dependencies.

### Compile
You will build all the Operator binaries and Docker images by running:

    make all

This assumes you have Docker installed and running on your development host.

The project uses the golang dep package manager to vendor all the golang source dependencies into the *vendor* directory.  You typically don't need to run any *dep* commands unless you are adding new golang package dependencies into the project outside of what is within the project for a given release.

After a full compile, you will have a *pgo* binary in $HOME/odev/bin and the Operator images in your local Docker registry.

### Release
You can perform a release build by running:

    make release

This will compile the Mac and Windows versions of *pgo*.


### Deploy
Now that you have built the Operator images, you can push them to your Kubernetes cluster if that cluster is remote to your development host.

You would then run:
```
make deployoperator
```

To deploy the Operator on your Kubernetes cluster.  If your Kubernetes cluster is not local to your development host, you will need to specify a Kube config file that will connect you to your Kubernetes cluster, see the Kube docs for details.


### Debug

Debug level logging in turned on by default when deploying the Operator.

You can view the REST API logs with the following alias:
```
alias alog='kubectl logs `kubectl get pod --selector=name=postgres-operator -o jsonpath="{.items[0].metadata.name}"` -c apiserver'
```

You can view the Operator core logic logs with the following alias:
```
alias olog='kubectl logs `kubectl get pod --selector=name=postgres-operator -o jsonpath="{.items[0].metadata.name}"` -c operator'
```
You can view the Scheduler logs with the following alias:
```
alias slog='kubectl logs `kubectl get pod --selector=name=postgres-operator -o jsonpath="{.items[0].metadata.name}"` -c scheduler'
```

You can enable the *pgo* CLI debugging with the following flag:
```
pgo version --debug
```

You can set the REST API URL as follows after a deployment if you are 
developing on your local host:
```
alias setip='export CO_APISERVER_URL=https://`kubectl get service postgres-operator -o=jsonpath="{.spec.clusterIP}"`:8443'
```
