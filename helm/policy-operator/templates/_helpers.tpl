{{/*
Expand the name of the chart.
*/}}
{{- define "policy-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "policy-operator.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart label.
*/}}
{{- define "policy-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "policy-operator.labels" -}}
helm.sh/chart: {{ include "policy-operator.chart" . }}
{{ include "policy-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "policy-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "policy-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
ServiceAccount name.
*/}}
{{- define "policy-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.name }}
{{- .Values.serviceAccount.name }}
{{- else }}
{{- printf "%s-controller-manager" (include "policy-operator.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Namespace name.
*/}}
{{- define "policy-operator.namespace" -}}
{{- .Values.namespace.name | default "policy-operator-system" }}
{{- end }}

{{/*
ConfigMap name for envVars auth mode.
*/}}
{{- define "policy-operator.configMapName" -}}
{{- if .Values.auth.envVars.configMapName }}
{{- .Values.auth.envVars.configMapName }}
{{- else }}
{{- printf "%s-azure-config" (include "policy-operator.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Secret name for envVars auth mode.
*/}}
{{- define "policy-operator.secretName" -}}
{{- if .Values.auth.envVars.secretName }}
{{- .Values.auth.envVars.secretName }}
{{- else }}
{{- printf "%s-azure-secret" (include "policy-operator.fullname" .) }}
{{- end }}
{{- end }}
