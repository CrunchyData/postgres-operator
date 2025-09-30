# Copyright 2025 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0
#
# This [jq] filter modifies a Kubernetes CustomResourceDefinition.
# Use the controller-gen "+kubebuilder:title" marker to identify schemas that need special manipulation.
#
# [jq]: https://jqlang.org

# merge recursively combines a stream of objects.
# https://jqlang.org/manual#multiplication-division-modulo
def merge(stream): reduce stream as $i ({}; . * $i);

# https://pkg.go.dev/k8s.io/api/core/v1#ImageVolumeSource
reduce paths(try .title == "$corev1.ImageVolumeSource") as $path (.;
  getpath($path) as $schema |
  setpath($path; $schema * {
    required: (["reference"] + ($schema.required // []) | sort),
    properties: {
      pullPolicy: { enum: ["Always", "Never", "IfNotPresent"] },
      reference: { minLength: 1 }
    }
  } | del(.title))
) |

# Kubernetes assumes the evaluation cost of an enum value is very large: https://issue.k8s.io/119511
# Look at every schema that has a populated "enum" property.
reduce paths(try .enum | length > 0) as $path (.;
  getpath($path) as $schema |
  setpath($path; $schema + { maxLength: ($schema.enum | map(length) | max) })
) |

# Kubernetes does not consider "allOf" when estimating CEL cost: https://issue.k8s.io/134029
# controller-gen might produce "allOf" when combining markers:
# https://github.com/kubernetes-sigs/controller-tools/issues/1270
#
# This (partially) addresses both by keeping only the smallest max, largest min, etc.
# Look at every schema that has a populated "allOf" property.
reduce paths(try .allOf | length > 0) as $path (.;
  (
    getpath($path) | merge(
      .,

      ( [.allOf[], .] | map({ minItems: (.minItems // empty) }) | max ) // empty,
      ( [.allOf[], .] | map({ maxItems: (.maxItems // empty) }) | min ) // empty,

      ( [.allOf[], .] | map({ minLength: (.minLength // empty) }) | max ) // empty,
      ( [.allOf[], .] | map({ maxLength: (.maxLength // empty) }) | min ) // empty,

      ( [.allOf[], .] | map({ minProperties: (.minProperties // empty) }) | max ) // empty,
      ( [.allOf[], .] | map({ maxProperties: (.maxProperties // empty) }) | min ) // empty,

      # NOTE: minimum and maximum must consider exclusiveMinimum/Maximum.
      empty
    )
  ) as $schema |

  # Remove "allOf" when it is entirely represented on $schema.
  if all($schema.allOf[]; keys - ($schema | keys) == []) then
    setpath($path; $schema | del(.allOf))
  else
    setpath($path; $schema)
  end
) |

# https://docs.k8s.io/tasks/extend-kubernetes/custom-resources/custom-resource-definitions#intorstring
#
# Remove "anyOf" from "x-kubernetes-int-or-string" schemas.
# The latter implies the former, and this makes CRDs about 1% smaller.
#
# This started as an attempt to work around https://issue.k8s.io/130946
reduce paths(try .["x-kubernetes-int-or-string"] == true) as $path (.;
  getpath($path) as $schema |

  if $schema.anyOf == [{ type: "integer" },{ type: "string" }] then
    setpath($path; $schema | del(.anyOf))
  end
) |

# Rename Kubebuilder annotations and move them to the top-level.
# The caller can turn these into YAML comments.
. += (.metadata.annotations | with_entries(select(.key | startswith("controller-gen.kubebuilder")) | .key = "# \(.key)")) |
.metadata.annotations |= with_entries(select(.key | startswith("controller-gen.kubebuilder") | not)) |

# Remove nulls and empty objects from metadata.
# Some very old generators would set a null creationTimestamp.
#
# https://github.com/kubernetes-sigs/controller-tools/issues/402
# https://issue.k8s.io/67610
del(.metadata | .. | select(length == 0)) |

# Remove status to avoid conflicts with the CRD controller.
# Some very old generators would set this field.
#
# https://github.com/kubernetes-sigs/controller-tools/issues/456
del(.status) |

.
