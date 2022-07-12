package main

var replacements = map[string]string{
	`HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_CONTAINER_RESOURCES: ""`: `{{- toYaml .Values.controllerManager.resources | nindent 10 }}`,
	`HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_NODESELECTOR: ""`:        `{{- toYaml .Values.controllerManager.nodeSelector | nindent 8 }}`,
	`HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_TOLERATIONS: ""`:         `{{- toYaml .Values.controllerManager.tolerations | nindent 8 }}`,
	`HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_AFFINITY: ""`:            `{{- toYaml .Values.controllerManager.affinity | nindent 8 }}`,
	`- HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_SCANNER_ARGS`:          `{{- if .Values.scanner.image.args }}{{- range .Values.scanner.image.args }}{{ nindent 8 "- --scanner-arg=" }}{{ . }}{{- end -}}{{ end }}`,
	`- HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_ERASER_ARGS`:           `{{- if .Values.eraser.image.args }}{{- range .Values.eraser.image.args }}{{ nindent 8 "- --eraser-arg=" }}{{ . }}{{- end -}}{{ end }}`,
	`- HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_COLLECTOR_ARGS`:        `{{- if .Values.collector.image.args }}{{- range .Values.collector.image.args }}{{ nindent 8 "- --collector-arg=" }}{{ . }}{{- end -}}{{ end }}`,
	`- HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_ADDITIONAL_ARGS`:       `{{- if .Values.controllerManager.image.additionalArgs }}{{- range .Values.controllerManager.image.additionalArgs }}{{ nindent 8 "- " }}{{ . }}{{- end -}}{{ end }}`,
}
