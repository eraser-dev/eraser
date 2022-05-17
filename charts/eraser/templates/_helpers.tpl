{{/*
Return the name of the chart. Use Values.nameOverride but if null use Chart.Name
*/}}
{{- define "eraser.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "eraser.serviceAccountName" -}}
{{- default (include "eraser.name" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
