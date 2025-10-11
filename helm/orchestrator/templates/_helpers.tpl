{{/*
Expand the name of the chart.
*/}}
{{- define "orchestrator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "orchestrator.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "orchestrator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "orchestrator.labels" -}}
helm.sh/chart: {{ include "orchestrator.chart" . }}
{{ include "orchestrator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "orchestrator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "orchestrator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "orchestrator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "orchestrator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Probe agent name
*/}}
{{- define "orchestrator.probeAgent.name" -}}
{{- printf "%s-probe-agent" (include "orchestrator.name" .) }}
{{- end }}

{{/*
Probe agent fullname
*/}}
{{- define "orchestrator.probeAgent.fullname" -}}
{{- printf "%s-probe-agent" (include "orchestrator.fullname" .) }}
{{- end }}

{{/*
Probe agent labels
*/}}
{{- define "orchestrator.probeAgent.labels" -}}
helm.sh/chart: {{ include "orchestrator.chart" . }}
{{ include "orchestrator.probeAgent.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Probe agent selector labels
*/}}
{{- define "orchestrator.probeAgent.selectorLabels" -}}
app.kubernetes.io/name: {{ include "orchestrator.probeAgent.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
