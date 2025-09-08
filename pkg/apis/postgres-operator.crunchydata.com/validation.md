<!--
# Copyright 2025 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0
-->

# Custom Resource Definitions

These directories contain Go types that serve as [DTO]s for communicating with the Kubernetes API.
We use the [controller-gen] tool to produce [CRD]s with schemas that match the Go types.
The CRD schema tells Kubernetes what fields and values are allowed in our API objects and how to handle changes to values.

> [!TIP]
> The CRD schema is *most* of the UX of our API

CRD schemas are modified OpenAPI 3.0 [validation] schemas.
Much of the schema defines what fields, types, and values are *allowed*.
`controller-gen` considers the [Go type] of a field and its [validation markers] for this.

Kubernetes uses its own algorithm to consider and accept changes to API objects: [Server-Side Apply], SSA.
CRD schemas contain non-standard attributes that affect SSA.
`controller-gen` considers the [processing markers] of a field for this.

🤔 Validation is *what* and SSA/processing is *how* 🤔

[controller-gen]: https://book.kubebuilder.io/reference/controller-gen
[CRD]: https://docs.k8s.io/tasks/extend-kubernetes/custom-resources/custom-resource-definitions
[DTO]: https://martinfowler.com/eaaCatalog/dataTransferObject.html
[Go type]: https://go.dev/ref/spec#Types
[Kubernetes API]: https://docs.k8s.io/concepts/overview/kubernetes-api
[processing markers]: https://book.kubebuilder.io/reference/markers/crd-processing
[Server-Side Apply]: https://docs.k8s.io/reference/using-api/server-side-apply
[validation]: https://docs.k8s.io/tasks/extend-kubernetes/custom-resources/custom-resource-definitions#validation
[validation markers]: https://book.kubebuilder.io/reference/markers/crd-validation


# OpenAPI Properties

Kubernetes supports a subset of the OpenAPI schema definition, which is a subset of an old draft of [JSON schema].

Fields can have these properties:

- `enum` is a list of values allowed in the field.
- `required` is a boolean indicating the field must be present in whatever payload.
  This does *NOT* indicate anything about the value being `null`, "zero," or "empty."
- `nullable` is a boolean indicating if the value can be `null`.
  When this is omitted or `false`, a `null` value is removed from whatever payload or replaced with the field's [default value].
- `type` is one of `integer`, `number`, `string`, `boolean`, `array`, or `object`.
- `format` constrains the `type` a bit further.
  It affects how `kubectl` presents the field and how the value turns up in CEL validation.

Numeric fields can have these properties:

- `minimum` and `maximum` are the smallest and largest numbers allowed in the field.
- `exclusiveMinimum` and `exclusiveMaximum` are booleans that indicate if the exact values above are allowed.
  That is, `true` here changes min and max from ≥ and ≤ to > and <.
- `multipleOf` is a number. Values in the field must be nicely divisible by this number.

String fields can have these properties:

- `minLength` and `maxLength` are the smallest and largest number of characters allowed in the field.
- `pattern` is a regular expression describing values allowed in the field.
  In Kubernetes, this is an [RE2] expression, *NOT* an ECMA expression.

Array fields can have these properties:

- `items` is a schema that describes what values are allowed inside the field.
- `minItems` and `maxItems` are the smallest and largest number of items allowed in the field.

The value of an object field is an unordered set of key/value pairs; a JSON object, a YAML mapping.
Kubernetes supports only two ways to treat the keys of those values: known or unknown.

The `properties` property indicates that the keys are known; these fields can have these properties:

- `properties` is an unordered set of key/value pairs (JSON object, YAML mapping) in which:
  - the key is the name of a field allowed in the object
  - the value is a schema that describes what values are allowed in that field.

The `additionalProperties` property indicates that the keys are unknown; these fields can have these properties:

- `additionalProperties` is a schema that describes what values are allowed in every... key-value value of the field.
- `minProperties` and `maxProperties` are the smallest and largest number of key-value keys allowed in the field.

> [!TIP]
> `properties` is like a Go struct // `additionalProperties` is like a Go map

[default value]: https://docs.k8s.io/tasks/extend-kubernetes/custom-resources/custom-resource-definitions#defaulting-and-nullable
[JSON schema]: https://json-schema.org/draft-06
[RE2]: https://github.com/google/re2#syntax


# CEL Rules

> [!IMPORTANT]
> When possible, use [OpenAPI properties](#FIXME) rather than CEL rules.
> The former do not affect the CRD [validation budget](#FIXME). <!-- https://imgur.com/CzpJn3j -->

