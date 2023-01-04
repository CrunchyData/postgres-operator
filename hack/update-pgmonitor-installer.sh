#!/usr/bin/env bash

# Copyright 2022 - 2023 Crunchy Data Solutions, Inc.
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

# This script updates the Kustomize installer for monitoring with the latest Grafana,
# Prometheus and Alert Manager configuration per the pgMonitor tag specified

directory=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

# The pgMonitor tag to use to refresh the current monitoring installer
pgmonitor_tag=v4.8.0

# Set the directory for the monitoring Kustomize installer
pgo_examples_monitoring_dir="${directory}/../../postgres-operator-examples/kustomize/monitoring"

# Create a tmp directory for checking out the pgMonitor tag
tmp_dir="${directory}/pgmonitor_tmp/"
mkdir -p "${tmp_dir}"

# Clone the pgMonitor repo and checkout the tag provided
git -C "${tmp_dir}" clone https://github.com/CrunchyData/pgmonitor.git
cd "${tmp_dir}/pgmonitor"
git checkout "${pgmonitor_tag}"

# Deviation from pgMonitor default!
# Update "${DS_PROMETHEUS}" to "PROMETHEUS" in all containers dashboards
find "grafana/containers" -type f -exec \
    sed -i 's/${DS_PROMETHEUS}/PROMETHEUS/' {} \; 
# Copy Grafana dashboards for containers
cp -r "grafana/containers/." "${pgo_examples_monitoring_dir}/config/grafana/dashboards"

# Deviation from pgMonitor default!
# Update the dashboard location to the default for the Grafana container.
sed -i 's#/etc/grafana/crunchy_dashboards#/etc/grafana/provisioning/dashboards#' \
    "grafana/linux/crunchy_grafana_dashboards.yml"
cp "grafana/linux/crunchy_grafana_dashboards.yml" "${pgo_examples_monitoring_dir}/config/grafana"

# Deviation from pgMonitor default!
# Update the URL for the Grafana data source configuration to use env vars for the Prometheus host
# and port.
sed -i 's#localhost:9090#$PROM_HOST:$PROM_PORT#' \
    "grafana/common/crunchy_grafana_datasource.yml"
cp "grafana/common/crunchy_grafana_datasource.yml" "${pgo_examples_monitoring_dir}/config/grafana"

# Deviation from pgMonitor default!
# Update the URL for the Grafana data source configuration to use env vars for the Prometheus host
# and port.
cp "prometheus/containers/crunchy-prometheus.yml.containers" "prometheus/containers/crunchy-prometheus.yml"
cat << EOF >> prometheus/containers/crunchy-prometheus.yml
alerting:
  alertmanagers:
  - scheme: http
    static_configs:
    - targets:
      - "crunchy-alertmanager:9093"
EOF
cp "prometheus/containers/crunchy-prometheus.yml" "${pgo_examples_monitoring_dir}/config/prometheus"

# Copy the default Alert Manager configuration
cp "alertmanager/common/crunchy-alertmanager.yml" "${pgo_examples_monitoring_dir}/config/alertmanager"
cp "prometheus/containers/alert-rules.d/crunchy-alert-rules-pg.yml.containers.example" \
    "${pgo_examples_monitoring_dir}/config/alertmanager/crunchy-alert-rules-pg.yml"

# Cleanup any temporary resources
rm -rf "${tmp_dir}"
