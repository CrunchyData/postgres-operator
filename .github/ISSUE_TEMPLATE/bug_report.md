---
name: Report a Bug
about: Found an issue? Let us fix it.
---

Please ensure you do the following when reporting a bug:

- [ ] Provide a concise description of what the bug is.
- [ ] Provide information about your environment.
- [ ] Provide clear steps to reproduce the bug.
- [ ] Attach applicable logs. Please do not attach screenshots showing logs unless you are unable to copy and paste the log data.
- [ ] Ensure any code / output examples are [properly formatted](https://docs.github.com/en/github/writing-on-github/basic-writing-and-formatting-syntax#quoting-code) for legibility.

Note that some logs needed to troubleshoot may be found in the `/pgdata/<CLUSTERNAME>/pg_log` directory on your Postgres instance.

An incomplete bug report can lead to delays in resolving the issue or the closing of a ticket, so please be as detailed as possible.

If you are looking for [general support](https://access.crunchydata.com/documentation/postgres-operator/latest/support/), please view the [support](https://access.crunchydata.com/documentation/postgres-operator/latest/support/) page for where you can ask questions.

Thanks for reporting the issue, we're looking forward to helping you!

## Overview

Add a concise description of what the bug is.

## Environment

Please provide the following details:

- Platform: (`Kubernetes`, `OpenShift`, `Rancher`, `GKE`, `EKS`, `AKS` etc.)
- Platform Version: (e.g. `1.20.3`, `4.7.0`)
- PGO Image Tag: (e.g. `centos8-4.7.0`)
- Postgres Version (e.g. `13`)
- Storage: (e.g. `hostpath`, `nfs`, or the name of your storage class)

## Steps to Reproduce

### REPRO

Provide steps to get to the error condition:

1. Run `...`
1. Do `...`
1. Try `...`

### EXPECTED

1. Provide the behavior that you expected.

### ACTUAL

1. Describe what actually happens

## Logs

Please provided appropriate log output or any configuration files that may help troubleshoot the issue. **DO NOT** include sensitive information, such as passwords.

## Additional Information

Please provide any additional information that may be helpful.
