---
title: "Install \"pgo\" Client"
date:
draft: false
weight: 30
---

# Install the PostgreSQL Operator (`pgo`) Client

The following will install and configure the `pgo` client on all systems.  For the
purpose of these instructions it's assumed that PGO: the Postgres Operator from Crunchy
Data is already deployed.

## Prerequisites

* For Kubernetes deployments: [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) configured to communicate with Kubernetes
* For OpenShift deployments: [oc](https://docs.openshift.com/container-platform/3.11/cli_reference/get_started_cli.html) configured to communicate with OpenShift

To authenticate with the PGO API:

* Client CA Certificate
* Client TLS Certificate
* Client Key
* `pgouser` file containing `<username>:<password>`

All of the requirements above should be obtained from an administrator who installed PGO.

## Linux and macOS

The following will setup the `pgo` client to be used on a Linux or macOS system.

### Installing the Client

First, download the `pgo` client from the
[GitHub official releases](https://github.com/CrunchyData/postgres-operator/releases). Crunchy Enterprise Customers can download the pgo binaries from https://access.crunchydata.com/ on the downloads page.

Next, install `pgo` in `/usr/local/bin` by running the following:

```bash
sudo mv /PATH/TO/pgo /usr/local/bin/pgo
sudo chmod +x /usr/local/bin/pgo
```

Verify the `pgo` client is accessible by running the following in the terminal:

```bash
pgo --help
```

#### Configuring Client TLS

With the client TLS requirements satisfied we can setup `pgo` to use them.

First, create a directory to hold these files by running the following command:

```bash
mkdir ${HOME?}/.pgo
chmod 700 ${HOME?}/.pgo
```

Next, copy the certificates to this new directory:

```bash
cp /PATH/TO/client.crt ${HOME?}/.pgo/client.crt && chmod 600 ${HOME?}/.pgo/client.crt
cp /PATH/TO/client.key ${HOME?}/.pgo/client.key && chmod 400 ${HOME?}/.pgo/client.key
```

Finally, set the following environment variables to point to the client TLS files:

```bash
cat <<EOF >> ${HOME?}/.bashrc
export PGO_CA_CERT="${HOME?}/.pgo/client.crt"
export PGO_CLIENT_CERT="${HOME?}/.pgo/client.crt"
export PGO_CLIENT_KEY="${HOME?}/.pgo/client.key"
EOF
```

Apply those changes to the current session by running:

```bash
source ~/.bashrc
```

#### Configuring `pgouser`

The `pgouser` file contains the username and password used for authentication with the Crunchy
PostgreSQL Operator.

To setup the `pgouser` file, run the following:

```bash
echo "<USERNAME_HERE>:<PASSWORD_HERE>" > ${HOME?}/.pgo/pgouser
```

```bash
cat <<EOF >> ${HOME?}/.bashrc
export PGOUSER="${HOME?}/.pgo/pgouser"
EOF
```

Apply those changes to the current session by running:

```bash
source ${HOME?}/.bashrc
```

#### Configuring the API Server URL

If the Crunchy PostgreSQL Operator is not accessible outside of the cluster, it's required
to setup a port-forward tunnel using the `kubectl` or `oc` binary.

In a separate terminal we need to setup a port forward to the Crunchy PostgreSQL Operator to
ensure connection can be made outside of the cluster:

```bash
# If deployed to Kubernetes
kubectl port-forward -n pgo svc/postgres-operator 8443:8443

# If deployed to OpenShift
oc port-forward -n pgo svc/postgres-operator 8443:8443
```

In the above examples, you can substitute `pgo` for the namespace that you
deployed the PostgreSQL Operator into.

**Note**: The port-forward will be required for the duration of using the
PostgreSQL client.

Next, set the following environment variable to configure the API server address:

```bash
cat <<EOF >> ${HOME?}/.bashrc
export PGO_APISERVER_URL="https://<IP_OF_OPERATOR_API>:8443"
EOF
```

**Note**: if port-forward is being used, the IP of the Operator API is `127.0.0.1`

Apply those changes to the current session by running:

```bash
source ${HOME?}/.bashrc
```

## PGO-Client Container

The following will setup the `pgo` client image in a Kubernetes or Openshift
environment. The image must be installed using the Ansible installer.

### Installing the PGO-Client Container
The pgo-client container can be installed with the Ansible installer by updating
the `pgo_client_container_install` variable in the inventory file. Set this
variable to true in the inventory file and run the ansible-playbook. As part of
the install the `pgo.tls` and `pgouser-<username>` secrets are used to configure
the `pgo` client.

### Using the PGO-Client Deployment
Once the container has been installed you can access it by exec'ing into the
pod. You can run single commands with the kubectl or oc command line tools
or multiple commands by exec'ing into the pod with bash.

```
kubectl exec -it -n pgo deploy/pgo-client -- pgo version

# or

kubectl exec -it -n pgo deploy/pgo-client bash
```

The deployment does not require any configuration to connect to the operator.

## Windows

The following will setup the `pgo` client to be used on a Windows system.

### Installing the Client

First, download the `pgo.exe` client from the
[GitHub official releases](https://github.com/CrunchyData/postgres-operator/releases).

Next, create a directory for `pgo` using the following:

* Left click the _Start_ button in the bottom left corner of the taskbar
* Type `cmd` to search for _Command Prompt_
* Right click the _Command Prompt_ application and click "Run as administrator"
* Enter the following command: `mkdir "%ProgramFiles%\postgres-operator"`

Within the same terminal copy the `pgo.exe` binary to the directory created above using the
following command:

```bash
copy %HOMEPATH%\Downloads\pgo.exe "%ProgramFiles%\postgres-operator"
```

Finally, add `pgo.exe` to the system path by running the following command in the terminal:

```bash
setx path "%path%;C:\Program Files\postgres-operator"
```

Verify the `pgo.exe` client is accessible by running the following in the terminal:

```bash
pgo --help
```

#### Configuring Client TLS

With the client TLS requirements satisfied we can setup `pgo` to use them.

First, create a directory to hold these files using the following:

* Left click the _Start_ button in the bottom left corner of the taskbar
* Type `cmd` to search for _Command Prompt_
* Right click the _Command Prompt_ application and click "Run as administrator"
* Enter the following command: `mkdir "%HOMEPATH%\pgo"`

Next, copy the certificates to this new directory:

```bash
copy \PATH\TO\client.crt "%HOMEPATH%\pgo"
copy \PATH\TO\client.key "%HOMEPATH%\pgo"
```

Finally, set the following environment variables to point to the client TLS files:

```bash
setx PGO_CA_CERT "%HOMEPATH%\pgo\client.crt"
setx PGO_CLIENT_CERT "%HOMEPATH%\pgo\client.crt"
setx PGO_CLIENT_KEY "%HOMEPATH%\pgo\client.key"
```

#### Configuring `pgouser`

The `pgouser` file contains the username and password used for authentication with the Crunchy
PostgreSQL Operator.

To setup the `pgouser` file, run the following:

* Left click the _Start_ button in the bottom left corner of the taskbar
* Type `cmd` to search for _Command Prompt_
* Right click the _Command Prompt_ application and click "Run as administrator"
* Enter the following command: `echo USERNAME_HERE:PASSWORD_HERE > %HOMEPATH%\pgo\pgouser`

Finally, set the following environment variable to point to the `pgouser` file:

```
setx PGOUSER "%HOMEPATH%\pgo\pgouser"
```

#### Configuring the API Server URL

If the Crunchy PostgreSQL Operator is not accessible outside of the cluster, it's required
to setup a port-forward tunnel using the `kubectl` or `oc` binary.

In a separate terminal we need to setup a port forward to the Crunchy PostgreSQL Operator to
ensure connection can be made outside of the cluster:

```bash
# If deployed to Kubernetes
kubectl port-forward -n pgo svc/postgres-operator 8443:8443

# If deployed to OpenShift
oc port-forward -n pgo svc/postgres-operator 8443:8443
```

In the above examples, you can substitute `pgo` for the namespace that you
deployed the PostgreSQL Operator into.

**Note**: The port-forward will be required for the duration of using the
PostgreSQL client.

Next, set the following environment variable to configure the API server address:

* Left click the _Start_ button in the bottom left corner of the taskbar
* Type `cmd` to search for _Command Prompt_
* Right click the _Command Prompt_ application and click "Run as administrator"
* Enter the following command: `setx PGO_APISERVER_URL "https://<IP_OF_OPERATOR_API>:8443"`
  * Note: if port-forward is being used, the IP of the Operator API is `127.0.0.1`

## Verify the Client Installation

After completing all of the steps above we can verify `pgo` is configured
properly by simply running the following:

```bash
pgo version
```

If the above command outputs versions of both the client and API server, the `pgo` client has been installed successfully.
