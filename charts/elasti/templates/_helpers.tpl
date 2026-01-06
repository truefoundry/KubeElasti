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
Namespace
*/}}
{{- define "global.namespace" }}
{{- default .Release.Namespace .Values.global.namespaceOverride }}
{{- end }}

{{/*
Global Labels
*/}}
{{- define "global.labels" }}
{{- if .Values.global.labels }}
{{- toYaml .Values.global.labels }}
{{- else }}
{}
{{- end }}
{{- end }}

{{/*
Global Annotations
*/}}
{{- define "global.annotations" }}
{{- if .Values.global.annotations }}
{{- toYaml .Values.global.annotations }}
{{- else }}
{}
{{- end }}
{{- end }}

{{/*
Service Account Annotations
*/}}
{{- define "global.serviceAccountAnnotations" -}}
{{- $merged := mergeOverwrite (deepCopy .Values.global.annotations) .Values.global.serviceAccount.annotations }}
{{- toYaml $merged }}
{{- end }}

{{/*
Return valid version label
*/}}
{{- define "elasti.versionLabelValue" -}}
{{ regexReplaceAll "[^-A-Za-z0-9_.]" (.Chart.AppVersion | default "unknown") "-" | trunc 63 | trimAll "-" | trimAll "_" | trimAll "." | quote }}
{{- end -}}

{{/*
Common labels - accepts dict with "context" and "name" parameters
Usage: include "elasti.labels" (dict "context" . "name" "elasti-operator")
*/}}
{{- define "elasti.labels" -}}
helm.sh/chart: {{ include "elasti.chart" .context }}
{{ include "elasti.selectorLabels" (dict "context" .context "name" .name) }}
app.kubernetes.io/managed-by: {{ .context.Release.Service }}
app.kubernetes.io/part-of: elasti
app.kubernetes.io/version: {{ include "elasti.versionLabelValue" .context }}
{{- with .context.Values.global.labels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
CRD labels - excludes global.labels to avoid instance-specific labels on cluster-scoped resources
*/}}
{{- define "elasti.crdLabels" -}}
helm.sh/chart: {{ include "elasti.chart" . }}
app.kubernetes.io/name: {{ include "elasti.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: elasti
app.kubernetes.io/version: {{ include "elasti.versionLabelValue" . }}
{{- end }}

{{/*
Selector labels - accepts dict with "context" and "name" parameters
Usage: include "elasti.selectorLabels" (dict "context" . "name" "elasti-operator")
*/}}
{{- define "elasti.selectorLabels" -}}
{{- if .name -}}
app.kubernetes.io/name: {{ .name }}
{{ end -}}
app.kubernetes.io/instance: {{ .context.Release.Name }}
{{- end }}

{{/*
Image pull secrets
*/}}
{{- define "elasti.imagePullSecrets" -}}
{{- toYaml .Values.global.imagePullSecrets }}
{{- end }}

{{/*
===========================================
Elasti Operator Helpers
===========================================
*/}}

{{/*
Expand the name of the elasti-operator component
*/}}
{{- define "elasti-operator.name" -}}
{{- default "elasti-operator" .Values.elastiController.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name for elasti-operator
*/}}
{{- define "elasti-operator.fullname" -}}
{{- if .Values.elastiController.fullnameOverride }}
{{- .Values.elastiController.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default "elasti-operator" .Values.elastiController.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Common labels for elasti-operator - uses global elasti.labels function
*/}}
{{- define "elasti-operator.labels" -}}
{{- include "elasti.labels" (dict "context" . "name" "elasti-operator") }}
{{- end }}

{{/*
Common labels - merges base labels and component-specific labels
Priority (highest to lowest): elastiController.commonLabels > base labels (which includes global.labels)
*/}}
{{- define "elasti-operator.commonLabels" -}}
{{- $baseLabels := include "elasti-operator.labels" . | fromYaml }}
{{- $mergedLabels := mergeOverwrite $baseLabels .Values.elastiController.commonLabels }}
{{- toYaml $mergedLabels }}
{{- end }}

{{/*
Common annotations - merges global.annotations with component-specific annotations
*/}}
{{- define "elasti-operator.commonAnnotations" -}}
{{- $merged := mergeOverwrite (deepCopy .Values.global.annotations) .Values.elastiController.commonAnnotations }}
{{- toYaml $merged }}
{{- end }}

{{/*
Control plane labels - returns the standard Kubernetes control-plane label for operator
*/}}
{{- define "elasti-operator.controlPlaneLabels" -}}
control-plane: controller-manager
{{- end }}

{{/*
Pod Labels - merges control plane labels, global and component labels, includes selector labels
Note: Using "elasti" for selector to maintain backward compatibility
*/}}
{{- define "elasti-operator.podLabels" -}}
{{- $controlPlaneLabels := include "elasti-operator.controlPlaneLabels" . | fromYaml }}
{{- $selectorLabels := include "elasti.selectorLabels" (dict "context" . "name" "elasti") | fromYaml }}
{{- $podLabels := mergeOverwrite (deepCopy .Values.global.podLabels) $controlPlaneLabels .Values.elastiController.podLabels $selectorLabels }}
{{- toYaml $podLabels }}
{{- end }}

{{/*
Pod Annotations - merges commonAnnotations with pod-specific annotations
Returns just the map (for merging with hardcoded pod annotations)
*/}}
{{- define "elasti-operator.podAnnotations" -}}
{{- $commonAnnotations := include "elasti-operator.commonAnnotations" . | fromYaml }}
{{- $podAnnotations := mergeOverwrite (deepCopy .Values.global.podAnnotations) $commonAnnotations .Values.elastiController.podAnnotations }}
{{- if $podAnnotations }}
{{- toYaml $podAnnotations }}
{{- end }}
{{- end }}

{{/*
Service Labels - merges control plane labels, commonLabels with service-specific labels
*/}}
{{- define "elasti-operator.serviceLabels" -}}
{{- $controlPlaneLabels := include "elasti-operator.controlPlaneLabels" . | fromYaml }}
{{- $commonLabels := include "elasti-operator.commonLabels" . | fromYaml }}
{{- $serviceLabels := mergeOverwrite (deepCopy .Values.global.serviceLabels) $controlPlaneLabels $commonLabels .Values.elastiController.service.labels }}
{{- toYaml $serviceLabels }}
{{- end }}

{{/*
Service Annotations - merges commonAnnotations with service-specific annotations
*/}}
{{- define "elasti-operator.serviceAnnotations" -}}
{{- $commonAnnotations := include "elasti-operator.commonAnnotations" . | fromYaml }}
{{- $serviceAnnotations := mergeOverwrite (deepCopy .Values.global.serviceAnnotations) $commonAnnotations .Values.elastiController.service.annotations }}
{{- toYaml $serviceAnnotations }}
{{- end }}

{{/*
Metrics Service Labels - merges control plane labels, commonLabels with metrics service-specific labels
*/}}
{{- define "elasti-operator.metricsServiceLabels" -}}
{{- $controlPlaneLabels := include "elasti-operator.controlPlaneLabels" . | fromYaml }}
{{- $commonLabels := include "elasti-operator.commonLabels" . | fromYaml }}
{{- $metricsServiceLabels := mergeOverwrite (deepCopy .Values.global.serviceLabels) $controlPlaneLabels $commonLabels .Values.elastiController.metricsService.labels }}
{{- toYaml $metricsServiceLabels }}
{{- end }}

{{/*
Metrics Service Annotations - merges commonAnnotations with metrics service-specific annotations
*/}}
{{- define "elasti-operator.metricsServiceAnnotations" -}}
{{- $commonAnnotations := include "elasti-operator.commonAnnotations" . | fromYaml }}
{{- $metricsServiceAnnotations := mergeOverwrite (deepCopy .Values.global.serviceAnnotations) $commonAnnotations .Values.elastiController.metricsService.annotations }}
{{- toYaml $metricsServiceAnnotations }}
{{- end }}

{{/*
Service Account Labels - merges commonLabels with service-account specific labels
*/}}
{{- define "elasti-operator.serviceAccountLabels" -}}
{{- $commonLabels := include "elasti-operator.commonLabels" . | fromYaml }}
{{- $serviceAccountLabels := mergeOverwrite (deepCopy .Values.global.serviceAccount.labels) $commonLabels .Values.elastiController.serviceAccount.labels }}
{{- toYaml $serviceAccountLabels }}
{{- end }}

{{/*
Service Account Annotations - merges commonAnnotations with service-account specific annotations
*/}}
{{- define "elasti-operator.serviceAccountAnnotations" -}}
{{- $commonAnnotations := include "elasti-operator.commonAnnotations" . | fromYaml }} 
{{- $serviceAccountAnnotations := mergeOverwrite (deepCopy .Values.global.serviceAccount.annotations) $commonAnnotations .Values.elastiController.serviceAccount.annotations }}
{{- toYaml $serviceAccountAnnotations }}
{{- end }}

{{/*
Deployment Labels - merges control plane labels, commonLabels with deployment-specific labels
*/}}
{{- define "elasti-operator.deploymentLabels" -}}
{{- $controlPlaneLabels := include "elasti-operator.controlPlaneLabels" . | fromYaml }}
{{- $commonLabels := include "elasti-operator.commonLabels" . | fromYaml }}
{{- $mergedLabels := mergeOverwrite (deepCopy .Values.global.deploymentLabels) $controlPlaneLabels $commonLabels .Values.elastiController.deploymentLabels }}
{{- toYaml $mergedLabels }}
{{- end }}

{{/*
Deployment annotations
*/}}
{{- define "elasti-operator.deploymentAnnotations" -}}
{{- $commonAnnotations := include "elasti-operator.commonAnnotations" . | fromYaml }}
{{- $mergedAnnotations := mergeOverwrite (deepCopy .Values.global.deploymentAnnotations) $commonAnnotations .Values.elastiController.deploymentAnnotations }}
{{- toYaml $mergedAnnotations }}
{{- end }}

{{/*
ServiceMonitor Labels - merges control plane labels, commonLabels with servicemonitor specific labels
*/}}
{{- define "elasti-operator.serviceMonitorLabels" -}}
{{- $controlPlaneLabels := include "elasti-operator.controlPlaneLabels" . | fromYaml }}
{{- $commonLabels := include "elasti-operator.commonLabels" . | fromYaml }}
{{- $serviceMonitorLabels := mergeOverwrite (deepCopy $controlPlaneLabels) $commonLabels .Values.elastiController.serviceMonitor.labels }}
{{- toYaml $serviceMonitorLabels }}
{{- end }}

{{/*
ServiceMonitor Annotations - merges commonAnnotations with servicemonitor specific annotations
*/}}
{{- define "elasti-operator.serviceMonitorAnnotations" -}}
{{- $commonAnnotations := include "elasti-operator.commonAnnotations" . | fromYaml }}
{{- $serviceMonitorAnnotations := mergeOverwrite (deepCopy $commonAnnotations) .Values.elastiController.serviceMonitor.annotations }}
{{- toYaml $serviceMonitorAnnotations }}
{{- end }}

{{/*
Selector Labels - merges control plane labels with base selector labels
*/}}
{{- define "elasti-operator.selectorLabels" -}}
{{- $controlPlaneLabels := include "elasti-operator.controlPlaneLabels" . | fromYaml }}
{{- $selectorLabels := include "elasti.selectorLabels" (dict "context" . "name" "elasti") | fromYaml }}
{{- $mergedLabels := mergeOverwrite $controlPlaneLabels $selectorLabels }}
{{- toYaml $mergedLabels }}
{{- end }}

{{/*
===========================================
Elasti Resolver Helpers
===========================================
*/}}

{{/*
Expand the name of the elasti-resolver component
*/}}
{{- define "elasti-resolver.name" -}}
{{- default "elasti-resolver" .Values.elastiResolver.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name for elasti-resolver
*/}}
{{- define "elasti-resolver.fullname" -}}
{{- if .Values.elastiResolver.fullnameOverride }}
{{- .Values.elastiResolver.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default "elasti-resolver" .Values.elastiResolver.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Common labels for elasti-resolver - uses global elasti.labels function
*/}}
{{- define "elasti-resolver.labels" -}}
{{- include "elasti.labels" (dict "context" . "name" "elasti-resolver") }}
{{- end }}

{{/*
Common labels - merges base labels and component-specific labels
Priority (highest to lowest): elastiResolver.commonLabels > base labels (which includes global.labels)
*/}}
{{- define "elasti-resolver.commonLabels" -}}
{{- $baseLabels := include "elasti-resolver.labels" . | fromYaml }}
{{- $mergedLabels := mergeOverwrite $baseLabels .Values.elastiResolver.commonLabels }}
{{- toYaml $mergedLabels }}
{{- end }}

{{/*
Common annotations - merges global.annotations with component-specific annotations
*/}}
{{- define "elasti-resolver.commonAnnotations" -}}
{{- $merged := mergeOverwrite (deepCopy .Values.global.annotations) .Values.elastiResolver.commonAnnotations }}
{{- toYaml $merged }}
{{- end }}

{{/*
App labels - returns the app identifier label for resolver
*/}}
{{- define "elasti-resolver.appLabels" -}}
app: elasti-resolver
{{- end }}

{{/*
Pod Labels - merges app labels, global and component labels, includes selector labels
Note: Using "elasti" for selector to maintain backward compatibility
*/}}
{{- define "elasti-resolver.podLabels" -}}
{{- $appLabels := include "elasti-resolver.appLabels" . | fromYaml }}
{{- $selectorLabels := include "elasti.selectorLabels" (dict "context" . "name" "elasti") | fromYaml }}
{{- $podLabels := mergeOverwrite (deepCopy .Values.global.podLabels) $appLabels .Values.elastiResolver.podLabels $selectorLabels }}
{{- toYaml $podLabels }}
{{- end }}

{{/*
Pod Annotations - merges commonAnnotations with pod-specific annotations
Returns just the map (for merging with hardcoded pod annotations)
*/}}
{{- define "elasti-resolver.podAnnotations" -}}
{{- $commonAnnotations := include "elasti-resolver.commonAnnotations" . | fromYaml }}
{{- $podAnnotations := mergeOverwrite (deepCopy .Values.global.podAnnotations) $commonAnnotations .Values.elastiResolver.podAnnotations }}
{{- if $podAnnotations }}
{{- toYaml $podAnnotations }}
{{- end }}
{{- end }}

{{/*
Service Labels - merges app labels, commonLabels with service-specific labels
*/}}
{{- define "elasti-resolver.serviceLabels" -}}
{{- $appLabels := include "elasti-resolver.appLabels" . | fromYaml }}
{{- $commonLabels := include "elasti-resolver.commonLabels" . | fromYaml }}
{{- $serviceLabels := mergeOverwrite (deepCopy .Values.global.serviceLabels) $appLabels $commonLabels .Values.elastiResolver.service.labels }}
{{- toYaml $serviceLabels }}
{{- end }}

{{/*
Service Annotations - merges commonAnnotations with service-specific annotations
*/}}
{{- define "elasti-resolver.serviceAnnotations" -}}
{{- $commonAnnotations := include "elasti-resolver.commonAnnotations" . | fromYaml }}
{{- $serviceAnnotations := mergeOverwrite (deepCopy .Values.global.serviceAnnotations) $commonAnnotations .Values.elastiResolver.service.annotations }}
{{- toYaml $serviceAnnotations }}
{{- end }}

{{/*
Service Account Labels - merges commonLabels with service account-specific labels
*/}}
{{- define "elasti-resolver.serviceAccountLabels" -}}
{{- $commonLabels := include "elasti-resolver.commonLabels" . | fromYaml }}
{{- $serviceAccountLabels := mergeOverwrite (deepCopy .Values.global.serviceAccount.labels) $commonLabels .Values.elastiResolver.serviceAccount.labels }}
{{- toYaml $serviceAccountLabels }}
{{- end }}

{{/*
Service Account Annotations - merges commonAnnotations with service account-specific annotations
*/}}
{{- define "elasti-resolver.serviceAccountAnnotations" -}}
{{- $commonAnnotations := include "elasti-resolver.commonAnnotations" . | fromYaml }} 
{{- $serviceAccountAnnotations := mergeOverwrite (deepCopy .Values.global.serviceAccount.annotations) $commonAnnotations .Values.elastiResolver.serviceAccount.annotations }}
{{- toYaml $serviceAccountAnnotations }}
{{- end }}

{{/*
Deployment Labels - merges app labels, commonLabels with deployment-specific labels
*/}}
{{- define "elasti-resolver.deploymentLabels" -}}
{{- $appLabels := include "elasti-resolver.appLabels" . | fromYaml }}
{{- $commonLabels := include "elasti-resolver.commonLabels" . | fromYaml }}
{{- $deploymentLabels := mergeOverwrite (deepCopy .Values.global.deploymentLabels) $appLabels $commonLabels .Values.elastiResolver.deploymentLabels }}
{{- toYaml $deploymentLabels }}
{{- end }}

{{/*
Deployment annotations
*/}}
{{- define "elasti-resolver.deploymentAnnotations" -}}
{{- $commonAnnotations := include "elasti-resolver.commonAnnotations" . | fromYaml }}
{{- $deploymentAnnotations := mergeOverwrite (deepCopy .Values.global.deploymentAnnotations) $commonAnnotations .Values.elastiResolver.deploymentAnnotations }}
{{- toYaml $deploymentAnnotations }}
{{- end }}

{{/*
ServiceMonitor Labels - merges commonLabels with servicemonitor specific labels
*/}}
{{- define "elasti-resolver.serviceMonitorLabels" -}}
{{- $commonLabels := include "elasti-resolver.commonLabels" . | fromYaml }}
{{- $serviceMonitorLabels := mergeOverwrite (deepCopy $commonLabels) .Values.elastiResolver.serviceMonitor.labels }}
{{- toYaml $serviceMonitorLabels }}
{{- end }}

{{/*
ServiceMonitor Annotations - merges commonAnnotations with servicemonitor specific annotations
*/}}
{{- define "elasti-resolver.serviceMonitorAnnotations" -}}
{{- $commonAnnotations := include "elasti-resolver.commonAnnotations" . | fromYaml }}
{{- $serviceMonitorAnnotations := mergeOverwrite (deepCopy $commonAnnotations) .Values.elastiResolver.serviceMonitor.annotations }}
{{- toYaml $serviceMonitorAnnotations }}
{{- end }}

{{/*
Selector Labels - merges app labels with base selector labels
*/}}
{{- define "elasti-resolver.selectorLabels" -}}
{{- $appLabels := include "elasti-resolver.appLabels" . | fromYaml }}
{{- $selectorLabels := include "elasti.selectorLabels" (dict "context" . "name" "elasti") | fromYaml }}
{{- $mergedLabels := mergeOverwrite $appLabels $selectorLabels }}
{{- toYaml $mergedLabels }}
{{- end }}

{{/*
===========================================
Common Helpers
===========================================
*/}}

{{/*
Common env values
*/}}
{{- define "elasti.commonEnvValues" -}}
- name: KUBERNETES_CLUSTER_DOMAIN
  value: {{ .Values.global.kubernetesClusterDomain | quote }}
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
