{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "install.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Crunchy labels
*/}}
{{- define "install.crunchyLabels" -}}
postgres-operator.crunchydata.com/control-plane: {{ .Chart.Name }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "install.labels" -}}
helm.sh/chart: {{ include "install.chart" . }}
{{ include "install.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{ include "install.crunchyLabels" .}}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "install.selectorLabels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{ include "install.crunchyLabels" .}}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "install.serviceAccountName" -}}
{{ .Chart.Name }}
{{- end }}

{{/*
Create the name of the Role/ClusterRole to use
*/}}
{{- define "install.roleName" -}}
{{ .Chart.Name }}
{{- end }}

{{/*
Create the name of the RoleBinding/ClusterRoleBinding to use
*/}}
{{- define "install.roleBindingName" -}}
{{ .Chart.Name }}
{{- end }}

{{/*
Create the kind for rolebindings. Will be RoleBinding in single
namespace mode or ClusterRoleBinding by default.
*/}}
{{- define "install.roleBindingKind" -}}
{{- if .Values.singleNamespace -}}
RoleBinding
{{- else -}}
ClusterRoleBinding
{{- end }}
{{- end }}

{{/*
Create the kind for role. Will be Role in single
namespace mode or ClusterRole by default.
*/}}
{{- define "install.roleKind" -}}
{{- if .Values.singleNamespace -}}
Role
{{- else -}}
ClusterRole
{{- end }}
{{- end }}