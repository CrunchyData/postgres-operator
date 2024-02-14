#!/usr/bin/env bash

# Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -eu

declare -r directory="$1"

# NOTE(cbandy): `kustomize` v4.1 and `kubectl` v1.22 will be able to change the
# kind of a resource: https://pr.k8s.io/101120
ruby -r 'set' -r 'yaml' -e '
directory = ARGV[0]
roles = YAML.load_stream(IO.read(File.join(directory, "role.yaml")))
operator = roles.shift

abort "Expected the operator ClusterRole first!" unless operator and operator["kind"] == "ClusterRole"

# The client used by the controller sets up a cache and an informer for any GVK
# that it GETs. That informer needs the "watch" permission.
# - https://github.com/kubernetes-sigs/controller-runtime/issues/1249
# - https://github.com/kubernetes-sigs/controller-runtime/issues/1454
# TODO(cbandy): Move this into an RBAC marker when it can be configured on the Manager.
operator["rules"].each do |rule|
	verbs = rule["verbs"].to_set
	rule["verbs"] = verbs.add("watch").sort if verbs.intersect? Set["get", "list"]
end

# Combine the other parsed Roles into the ClusterRole.
rules = operator["rules"] + roles.flat_map { |role| role["rules"] }
rules = rules.
	group_by { |rule| rule.slice("apiGroups", "resources") }.
	map do |(group_resource, rules)|
		verbs = rules.flat_map { |rule| rule["verbs"] }.to_set.sort
		group_resource.merge("verbs" => verbs)
	end
operator["rules"] = rules.sort_by { |rule| rule.to_a }

# Combine resources that have the same verbs.
rules = operator["rules"].
	group_by { |rule| rule.slice("apiGroups", "verbs") }.
	map do |(group_verb, rules)|
		resources = rules.flat_map { |rule| rule["resources"] }.to_set.sort
		rule = group_verb.merge("resources" => resources)
		rule.slice("apiGroups", "resources", "verbs") # keep the keys in order
	end
operator["rules"] = rules.sort_by { |rule| rule.to_a }

operator["metadata"] = { "name" => "postgres-operator" }
IO.write(File.join(directory, "cluster", "role.yaml"), YAML.dump(operator))

operator["kind"] = "Role"
IO.write(File.join(directory, "namespace", "role.yaml"), YAML.dump(operator))
' -- "${directory}"
