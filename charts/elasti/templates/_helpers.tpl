{{/*
Expand the name of the chart.
*/}}
{{- define "elasti.name" -}}
{{- default .Chart.Name .Values.global.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "elasti.fullname" -}}
{{- if .Values.global.fullnameOverride }}
{{- .Values.global.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.global.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "elasti.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "elasti.labels" -}}
helm.sh/chart: {{ include "elasti.chart" . }}
{{ include "elasti.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "elasti.selectorLabels" -}}
app.kubernetes.io/name: {{ include "elasti.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Common env values
*/}}
{{- define "elasti.commonEnvValues" -}}
- name: KUBERNETES_CLUSTER_DOMAIN
  value: {{ .Values.global.kubernetesClusterDomain }}
- name: ELASTI_OPERATOR_NAMESPACE
  value: {{ .Release.Namespace }}
- name: ELASTI_OPERATOR_DEPLOYMENT_NAME
  value: {{ include "elasti.fullname" . }}-operator-controller-manager
- name: ELASTI_OPERATOR_SERVICE_NAME
  value: {{ include "elasti.fullname" . }}-operator-controller-service
- name: ELASTI_OPERATOR_PORT
  value: {{ .Values.elastiController.service.port | quote }}
- name: ELASTI_RESOLVER_NAMESPACE
  value: {{ .Release.Namespace }}
- name: ELASTI_RESOLVER_DEPLOYMENT_NAME
  value: {{ include "elasti.fullname" . }}-resolver
- name: ELASTI_RESOLVER_SERVICE_NAME
  value: {{ include "elasti.fullname" . }}-resolver-service
- name: ELASTI_RESOLVER_PORT
  value: {{ .Values.elastiResolver.service.port | quote }}
- name: ELASTI_RESOLVER_PROXY_PORT
  value: {{ .Values.elastiResolver.reverseProxyService.port | quote }}
{{- end }}
