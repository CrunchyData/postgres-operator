
This directory contains the files that are used to install [Crunchy PostgreSQL for Kubernetes][hub-listing],
which uses the PostgreSQL Operator, using [Operator Lifecycle Manager][OLM].

The integration centers around a [ClusterServiceVersion][olm-csv] [manifest](./postgresoperator.csv.yaml)
that gets packaged for OperatorHub. Changes there are accepted only if they pass all the [scorecard][]
tests. Consult the [technical requirements][hub-contrib] when making changes.

[hub-contrib]: https://github.com/operator-framework/community-operators/blob/master/docs/contributing.md
[hub-listing]: https://operatorhub.io/operator/postgresql
[olm-csv]: https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/building-your-csv.md
[OLM]: https://github.com/operator-framework/operator-lifecycle-manager
[scorecard]: https://github.com/operator-framework/operator-sdk/blob/master/doc/test-framework/scorecard.md
