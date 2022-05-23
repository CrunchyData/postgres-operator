---
name: Support
about: "Learn how to interact with the PGO community"
---

If you believe you have found have found a bug, please open up [Bug Report](https://github.com/CrunchyData/postgres-operator/issues/new?template=bug_report.md)

If you have a feature request, please open up a [Feature Request](https://github.com/CrunchyData/postgres-operator/issues/new?template=feature_request.md)

You can find information about general PGO [support](https://access.crunchydata.com/documentation/postgres-operator/latest/support/) at:

[https://access.crunchydata.com/documentation/postgres-operator/latest/support/](https://access.crunchydata.com/documentation/postgres-operator/latest/support/)

## Questions

For questions that are neither bugs nor feature requests, please be sure to

- [ ] Provide information about your environment (see below for more information).
- [ ] Provide any steps or other relevant details related to your question.
- [ ] Attach logs, where applicable. Please do not attach screenshots showing logs unless you are unable to copy and paste the log data.
- [ ] Ensure any code / output examples are [properly formatted](https://docs.github.com/en/github/writing-on-github/basic-writing-and-formatting-syntax#quoting-code) for legibility.

Besides Pod logs, logs may also be found in the `/pgdata/pg<MAJOR_VERSION>/log` directory on your Postgres instance.

If you are looking for [general support](https://access.crunchydata.com/documentation/postgres-operator/latest/support/), please view the [support](https://access.crunchydata.com/documentation/postgres-operator/latest/support/) page for where you can ask questions.

### Environment

Please provide the following details:

- Platform: (`Kubernetes`, `OpenShift`, `Rancher`, `GKE`, `EKS`, `AKS` etc.)
- Platform Version: (e.g. `1.20.3`, `4.7.0`)
- PGO Image Tag: (e.g. `ubi8-5.1.0-0`)
- Postgres Version (e.g. `14`)
- Storage: (e.g. `hostpath`, `nfs`, or the name of your storage class)
