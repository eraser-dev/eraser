{{/*
Return the name of the chart. Use Values.nameOverride but if null use Chart.Name
*/}}
{{- define "eraser.name" -}}
{{- default .Chart.Name .Values.nameOverride -}}
{{- end -}}