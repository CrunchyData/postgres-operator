<!--
# Copyright 2022 - 2024 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0
-->

# Definitions

[k8s-names]: https://docs.k8s.io/concepts/overview/working-with-objects/names/

### DNS subdomain

Most resource types require this kind of name. It must be 253 characters or less,
lowercase, and alphanumeric with hyphens U+002D and dots U+002E allowed in between.

- [k8s.io/apimachinery/pkg/util/validation.IsDNS1123Subdomain](https://pkg.go.dev/k8s.io/apimachinery/pkg/util/validation#IsDNS1123Subdomain)

### DNS label

Some resource types require this kind of name. It must be 63 characters or less,
lowercase, and alphanumeric with hyphens U+002D allowed in between.

Some have a stricter requirement to start with an alphabetic (nonnumerical) character.

- [k8s.io/apimachinery/pkg/util/validation.IsDNS1123Label](https://pkg.go.dev/k8s.io/apimachinery/pkg/util/validation#IsDNS1123Label)
- [k8s.io/apimachinery/pkg/util/validation.IsDNS1035Label](https://pkg.go.dev/k8s.io/apimachinery/pkg/util/validation#IsDNS1035Label)


# Labels

[k8s-labels]: https://docs.k8s.io/concepts/overview/working-with-objects/labels/

Label names must be 317 characters or less. The portion before an optional slash U+002F
must be a DNS subdomain. The portion after must be 63 characters or less.

Label values must be 63 characters or less and can be empty.

Both label names and values must be alphanumeric with hyphens U+002D, underscores U+005F,
and dots U+002E allowed in between.

- [k8s.io/apimachinerypkg/util/validation.IsQualifiedName](https://pkg.go.dev/k8s.io/apimachinery/pkg/util/validation#IsQualifiedName)
- [k8s.io/apimachinerypkg/util/validation.IsValidLabelValue](https://pkg.go.dev/k8s.io/apimachinery/pkg/util/validation#IsValidLabelValue)


# Annotations

[k8s-annotations]: https://docs.k8s.io/concepts/overview/working-with-objects/annotations/

Annotation names must be 317 characters or less. The portion before an optional slash U+002F
must be a DNS subdomain. The portion after must be 63 characters or less and alphanumeric with
hyphens U+002D, underscores U+005F, and dots U+002E allowed in between.

Annotation values may contain anything, but the combined size of *all* names and values
must be 256 KiB or less.

- [https://pkg.go.dev/k8s.io/apimachinery/pkg/api/validation.ValidateAnnotations](https://pkg.go.dev/k8s.io/apimachinery/pkg/api/validation#ValidateAnnotations)


# Specifics

The Kubernetes API validates custom resource metadata.
[Custom resource names are DNS subdomains](https://releases.k8s.io/v1.23.0/staging/src/k8s.io/apiextensions-apiserver/pkg/registry/customresource/validator.go#L60).
It may be possible to limit this further through validation. This is a stated
goal of [CEL expression validation](https://docs.k8s.io/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#validation-rules).

[ConfigMap names are DNS subdomains](https://releases.k8s.io/v1.23.0/pkg/apis/core/validation/validation.go#L5618).

[CronJob names are DNS subdomains](https://docs.k8s.io/concepts/workloads/controllers/cron-jobs/)
but must be [52 characters or less](https://releases.k8s.io/v1.23.0/pkg/apis/batch/validation/validation.go#L281).

[Deployment names are DNS subdomains](https://releases.k8s.io/v1.23.0/pkg/apis/apps/validation/validation.go#L632).

[Job names are DNS subdomains](https://releases.k8s.io/v1.23.0/pkg/apis/batch/validation/validation.go#L86).
When `.spec.completionMode = Indexed`, the name must be shorter (closer to 61 characters, it depends).
When `.spec.manualSelector` is unset, its Pods get (and must have) a "job-name" label, limiting the
name to 63 characters or less.

[Namespace names are DNS labels](https://releases.k8s.io/v1.23.0/pkg/apis/core/validation/validation.go#L5963).

[PersistentVolumeClaim (PVC) names are DNS subdomains](https://releases.k8s.io/v1.23.0/pkg/apis/core/validation/validation.go#L2066).

[Pod names are DNS subdomains](https://releases.k8s.io/v1.23.0/pkg/apis/core/validation/validation.go#L3443).
The strategy for [generating Pod names](https://releases.k8s.io/v1.23.0/pkg/registry/core/pod/strategy.go#L62) truncates to 63 characters.
The `.spec.hostname` field must be 63 characters or less.

PodDisruptionBudget (PDB)

[ReplicaSet names are DNS subdomains](https://releases.k8s.io/v1.23.0/pkg/apis/apps/validation/validation.go#L655).

Role

RoleBinding

[Secret names are DNS subdomains](https://releases.k8s.io/v1.23.0/pkg/apis/core/validation/validation.go#L5515).

[Service names are DNS labels](https://docs.k8s.io/concepts/services-networking/service/)
that must begin with a letter.

ServiceAccount (subdomain)

[StatefulSet names are DNS subdomains](https://docs.k8s.io/concepts/workloads/controllers/statefulset/),
but its Pods get [hostnames](https://releases.k8s.io/v1.23.0/pkg/apis/core/validation/validation.go#L3561)
so it must be shorter (closer to 61 characters, it depends). Its Pods also get a "controller-revision-hash"
label with [11 characters appended](https://issue.k8s.io/64023), limiting the name to 52 characters or less.

