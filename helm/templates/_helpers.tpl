{{/*
  _helpers.tpl

  This file defines helper template functions 
  that produce consistent naming conventions throughout your chart.
  They are referenced by other templates (e.g. Deployment, Service, etc.) 
  to ensure you don't have to repeat logic each time.

  we define three helpers:
  1. "reference-app-wms-go.name" — a base name for the chart
  2. "reference-app-wms-go.chart" — a string combining chart name and version
  3. "reference-app-wms-go.fullname" — a "full name" that can incorporate overrides
*/}}


{{/*
  This helper returns the base name of the chart. If `.Values.nameOverride` is set,
  that value is used. Otherwise, it defaults to `.Chart.Name`.
  We also truncate to 63 characters to avoid issues with max length in Kubernetes,
  and remove any trailing hyphens.
*/}}
{{- define "reference-app-wms-go.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}


{{/*
  This helper returns a combination of the chart name and the chart version,
  for labeling or other informational purposes.
  Example: "mychart-0.1.0"
*/}}
{{- define "reference-app-wms-go.chart" -}}
{{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}


{{/*
  This is the main helper that provides a "full name" for your resources.
  If `.Values.fullnameOverride` is specified, we use that (also truncated),
  or else we fall back to the result of "reference-app-wms-go.name".
*/}}

{{- define "reference-app-wms-go.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else }}
{{- include "reference-app-wms-go.name" . -}}
{{- end -}}
{{- end -}}
